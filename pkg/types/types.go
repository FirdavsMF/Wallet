package types

type Money int64

type Category string

type Status string

type PaymentCategory string

const (
	PaymentStatusOk         Status = "OK"
	PaymentStatusFail       Status = "FAIL"
	PaymentStatusInProgress Status = "INPROGRESS"
)

type Payment struct {
	ID        string
	Amount    Money
	Category  PaymentCategory
	Status    Status
	AccountID int64
}
type Phone string

type Account struct {
	ID      int64
	Phone   Phone
	Balance Money
}

type Favorite struct {
	ID        string
	AccountID int64
	Name      string
	Amount    Money
	Category  PaymentCategory
}