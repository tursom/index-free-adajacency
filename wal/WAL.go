package wal

type (
	WAL struct {
		rollbacks []func()
	}
)

func SetValue[T any](wal *WAL, p *T, new T) {
	AddValueRec(wal, p)
	*p = new
}

func IncInt(wal *WAL, p *int) {
	AddValueRec(wal, p)
	*p++
}

func DecInt(wal *WAL, p *int) {
	AddValueRec(wal, p)
	*p--
}

func IncUInt32(wal *WAL, p *uint32) {
	AddValueRec(wal, p)
	*p++
}

func DecUInt32(wal *WAL, p *uint32) {
	AddValueRec(wal, p)
	*p--
}

func AddValueRec[T any](wal *WAL, p *T) {
	oldValue := *p
	wal.AddRollBack(func() {
		*p = oldValue
	})
}

func (log *WAL) AddRollBack(rollback func()) {
	log.rollbacks = append(log.rollbacks, rollback)
}

func (log *WAL) RollBackWhenPanic(err any) {
	if err == nil {
		return
	}

	log.RollBack()

	panic(err)
}

func (log *WAL) RollBack() {
	for _, rollback := range log.rollbacks {
		rollback()
	}

	log.rollbacks = nil
}
