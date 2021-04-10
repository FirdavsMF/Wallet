package wallet

import (
	"testing"
)

func TestService_FindAccountByID_success(t *testing.T) {
	service := &Service{}
	service.RegisterAccount("+9929030232")

	account, err := service.FindAccountByID(1)
	if err != nil {
		t.Errorf("\ngot: %v \nwant: nil  ", account)
	}
}

func TestService_FindAccountByID_notFound(t *testing.T) {
	service := Service{}
	service.RegisterAccount("+9929030232")

	_, err := service.FindAccountByID(2)
	if err == nil {
		t.Error(ErrAccountNotFound)
		return
	}

}