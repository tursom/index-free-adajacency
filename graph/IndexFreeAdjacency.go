package graph

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"unsafe"
)

const (
	pageSize = 16
)

type (
	Graph struct {
		nodes         []slice[Node]
		relations     []slice[Relation]
		properties    []slice[property]
		freeNode      Index
		freeRelation  *Relation
		freeProperty  *property
		usedNodes     BitSet
		usedRelations BitSet
		nodeCount     int
		relationCount int
	}

	Index = int

	Node struct {
		// reused to next free node
		index         Index
		label         string
		firstProperty *property
		firstRelation *Relation
		g             *Graph
	}

	Relation struct {
		index    Index
		g        *Graph
		from, to *Node
		// s, start = from
		// e, end = to
		// sn as next when it's free
		sp, ep, sn, en *Relation
		firstProperty  *property
	}

	property struct {
		next  *property
		key   string
		value any
	}

	nodeIterator struct {
		node *Node
	}

	relationIterator struct {
		node     *Node
		relation *Relation
	}

	slice[T any] struct {
		arr *[pageSize]T
		len uint32
	}
)

var (
	ErrDeletedNode = fmt.Errorf("node alrady deleted")
	ErrRelation    = fmt.Errorf("node have relations")

	ErrDeletedRelation = fmt.Errorf("relation alrady deleted")
)

func (g *Graph) Nodes() Iterator[*Node] {
	firstUsed := g.usedNodes.NextUp(-1)
	var node *Node = nil
	if firstUsed >= 0 {
		node = g.getNodeUnsafe(firstUsed)
	}
	return &nodeIterator{node}
}

func (g *Graph) NodeCount() int {
	return g.nodeCount
}

func (g *Graph) GetNode(index Index) *Node {
	if index >= len(g.nodes)*pageSize || !g.usedNodes.Get(index) {
		return nil
	}

	return g.getNodeUnsafe(index)
}

func (g *Graph) RelationCount() int {
	return g.relationCount
}

func (g *Graph) getNodeUnsafe(index Index) *Node {
	return &g.nodes[index/pageSize].arr[index%pageSize]
}

func (g *Graph) GetRelation(index Index) *Relation {
	if index >= len(g.relations)*pageSize || !g.usedRelations.Get(index) {
		return nil
	}

	return g.getRelationUnsafe(index)
}

func (g *Graph) getRelationUnsafe(index Index) *Relation {
	return &g.relations[index/pageSize].arr[index%pageSize]
}

func (g *Graph) AddNode(label string) (index Index) {
	if g.freeNode != 0 {
		freeNodeIndex := g.freeNode - 1
		n := g.getNodeUnsafe(freeNodeIndex)
		g.freeNode = n.index
		n.index = freeNodeIndex
		index = freeNodeIndex
	} else {
		lastNodes := lastPage(&g.nodes)

		index = (len(g.nodes)-1)*pageSize + int(lastNodes.len)
		lastNodes.arr[lastNodes.len] = Node{
			index: index,
			g:     g,
		}
		lastNodes.len++
	}

	n := g.getNodeUnsafe(index)
	n.label = label

	if g.usedNodes.BitLength() < len(g.nodes)*pageSize {
		g.usedNodes = append(g.usedNodes, 0)
	}
	g.usedNodes.SetBit(index, true)

	g.nodeCount++

	return index
}

func (g *Graph) AddRelation(from, to Index) Index {
	f := g.GetNode(from)
	if f == nil {
		return -1
	}
	t := g.GetNode(to)
	if t == nil {
		return -1
	}

	var rla *Relation
	if g.freeRelation != nil {
		// reuse free Relation slot
		rla = g.freeRelation
		g.freeRelation = rla.sn

		rla.from = f
		rla.to = t
		rla.sp = nil
		rla.ep = nil
	} else {
		lastRelations := lastPage(&g.relations)
		lastRelations.arr[lastRelations.len] = Relation{
			index: (len(g.relations)-1)*pageSize + int(lastRelations.len),
			from:  f,
			to:    t,
		}

		rla = &lastRelations.arr[lastRelations.len]
		rla.g = g

		lastRelations.len++
	}

	rla.sn = f.firstRelation
	if f.firstRelation != nil {
		f.firstRelation.sp = rla
	}
	f.firstRelation = rla

	rla.en = t.firstRelation
	if t.firstRelation != nil {
		t.firstRelation.ep = rla
	}
	t.firstRelation = rla

	if g.usedRelations.BitLength() < len(g.relations)*pageSize {
		g.usedRelations = append(g.usedRelations, 0)
	}
	g.usedRelations.SetBit(rla.index, true)

	g.relationCount++

	return rla.index
}

func (g *Graph) DeleteNode(node Index) error {
	if node < 0 || !g.usedNodes.Get(node) {
		return ErrDeletedNode
	}

	n := g.getNodeUnsafe(node)
	if n.firstRelation != nil {
		return ErrRelation
	}

	g.usedNodes.SetBit(node, false)

	if n.firstProperty != nil {
		lastPpt := n.firstProperty
		for lastPpt.next != nil {
			lastPpt = lastPpt.next
		}

		lastPpt.next = g.freeProperty
		g.freeProperty = n.firstProperty
	}

	index := n.index
	n.index = g.freeNode
	g.freeNode = index + 1

	g.nodeCount--

	return nil
}

func (g *Graph) DeleteRelation(relation Index) error {
	if relation < 0 || !g.usedRelations.Get(relation) {
		return ErrDeletedRelation
	}

	g.usedRelations.SetBit(relation, false)

	r := g.getRelationUnsafe(relation)

	if r.sp == nil {
		r.from.firstRelation = r.sn
	} else if r.sp.from == r.from {
		r.sp.sn = r.sn
	} else {
		r.sp.en = r.sn
	}

	if r.ep == nil {
		r.to.firstRelation = r.en
	} else if r.ep.to == r.to {
		r.ep.en = r.en
	} else {
		r.ep.sn = r.en
	}

	if r.sn == nil { // safe check, do nothing
	} else if r.sn.from == r.from {
		r.sn.sp = r.sp
	} else {
		r.sn.ep = r.sp
	}

	if r.en == nil { // safe check, do nothing
	} else if r.en.to == r.to {
		r.en.ep = r.ep
	} else {
		r.en.sp = r.ep
	}

	r.sn = g.freeRelation
	g.freeRelation = r

	g.relationCount--

	return nil
}

func (g *Graph) CheckNodes() Index {
	for i := 0; i < g.nodeCount; i++ {
		node := g.getNodeUnsafe(i)
		if g.usedNodes.Get(i) {
			if node.index != i {
				return i
			}
		} else {
			if i == node.index-1 {
				return i
			}
		}
	}
	return -1
}

func (g *Graph) CheckRelations() Index {
	for relation := g.freeRelation; relation != nil; relation = relation.sn {
		if g.usedRelations.Get(relation.index) {
			return relation.index
		}
	}

	//TODO

	return -1
}

func (n *Node) String() string {
	var sb strings.Builder
	sb.WriteString("Node{label:\"")
	sb.WriteString(n.label)
	sb.WriteString("\", properties: ")
	bytes, _ := json.Marshal(n.GetProperties())
	sb.WriteString(string(bytes))
	sb.WriteString("}")
	return sb.String()
}

func (n *Node) Graph() *Graph {
	return n.g
}

func (n *Node) ID() int {
	return n.index
}

func (n *Node) Lable() string {
	return n.label
}

func (n *Node) GetProperties() map[string]any {
	return n.firstProperty.toMap()
}

func (n *Node) SetProperty(key string, value any) {
	setProperty(n.g, &n.firstProperty, key, value)
}

func (n *Node) DelProperty(key string) bool {
	return delProperty(n.g, &n.firstProperty, key)
}

func (n *Node) Relations() Iterator[*Relation] {
	return &relationIterator{n, n.firstRelation}
}

func (r *Relation) Index() Index {
	return r.index
}

func (r *Relation) Graph() *Graph {
	return r.g
}

func (r *Relation) From() *Node {
	return r.from
}

func (r *Relation) To() *Node {
	return r.to
}

func (r *Relation) Sp() *Relation {
	return r.sp
}

func (r *Relation) Ep() *Relation {
	return r.ep
}

func (r *Relation) Sn() *Relation {
	return r.sn
}

func (r *Relation) En() *Relation {
	return r.en
}

func (r *Relation) GetProperties() map[string]any {
	return r.firstProperty.toMap()
}

func (r *Relation) SetProperty(key string, value any) {
	setProperty(r.g, &r.firstProperty, key, value)
}

func (r *Relation) DelProperty(key string) bool {
	return delProperty(r.g, &r.firstProperty, key)
}

func (r *Relation) String() string {
	var sb strings.Builder
	sb.WriteString("Relation(")
	sb.WriteString(fmt.Sprintf("%d-->%d", r.from.ID(), r.to.ID()))
	bytes, _ := json.Marshal(r.GetProperties())
	sb.Write(bytes)
	sb.WriteString(")")
	return sb.String()
}

func (p *property) toMap() map[string]any {
	m := make(map[string]any)
	for p != nil {
		m[p.key] = p.value
		//goland:noinspection GoAssignmentToReceiver
		p = p.next
	}
	return m
}

func setProperty(g *Graph, p **property, key string, value any) {
	ppt := *p
	for ppt != nil {
		if ppt.key == key {
			ppt.value = value
			return
		}
		ppt = ppt.next
	}

	if g.freeProperty != nil {
		ppt = g.freeProperty
		g.freeProperty = ppt.next
	} else {
		propertiesPage := lastPage(&g.properties)
		ppt = &propertiesPage.arr[propertiesPage.len]
		propertiesPage.len++
	}

	ppt.next = *p
	*p = ppt

	ppt.key = key
	ppt.value = value
}

func delProperty(g *Graph, p **property, key string) bool {
	prev := p
	ppt := *prev
	for ppt != nil {
		if ppt.key == key {
			*prev = ppt.next

			ppt.next = g.freeProperty
			g.freeProperty = ppt

			return true
		}
		prev = &ppt.next
		ppt = *prev
	}
	return false
}

func (n *nodeIterator) HasNext() bool {
	return n.node != nil
}

func (n *nodeIterator) Next() *Node {
	node := n.node
	next := n.node.g.usedNodes.NextUp(n.node.index)
	if next < 0 {
		n.node = nil
	} else {
		n.node = n.node.g.getNodeUnsafe(next)
	}
	return node
}

func (r *relationIterator) HasNext() bool {
	return r.relation != nil
}

func (r *relationIterator) Next() *Relation {
	relation := r.relation
	if r.relation.from == r.node {
		r.relation = relation.sn
	} else {
		r.relation = relation.en
	}
	return relation
}

func indexOf[T any](s []T, v *T) int {
	begin := *(*uintptr)(unsafe.Pointer(&s))
	addr := uintptr(unsafe.Pointer(v))
	return int((addr - begin) / reflect.TypeOf(*v).Size())
}

func lastPage[T any](s *[]slice[T]) *slice[T] {
	if len(*s) == 0 {
		*s = append(*s, slice[T]{arr: new([pageSize]T)})
		return &(*s)[0]
	}
	if (*s)[len(*s)-1].len == pageSize {
		*s = append(*s, slice[T]{arr: new([pageSize]T)})
	}
	return &(*s)[len(*s)-1]
}
