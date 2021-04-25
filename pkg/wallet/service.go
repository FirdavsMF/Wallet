package wallet

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/FirdavsMF/Wallet/pkg/types"
	"github.com/google/uuid"
)

// распространённые возможные ошибки
var ErrPhoneRegistered = errors.New("phone already registred")
var ErrAmountMustBePositive = errors.New("amount must be greater than zero")
var ErrAccountNotFound = errors.New("account not found")
var ErrNotEnoughtBalance = errors.New("account not enough balance")
var ErrPaymentNotFound = errors.New("payment not found")
var ErrFavoriteNotFound = errors.New("favorite not found")
var ErrFileNotFound = errors.New("file not found")
var err error


type Service struct  {
	NextAccountID int64
	accounts []*types.Account 
	payments []*types.Payment
	favorites []*types.Favorite 
}


// RegisterAccount регистрирует  нового пользователя в системе
func (s *Service) RegisterAccount(phone types.Phone) (*types.Account, error) {
	for _, account := range s.accounts {
		if account.Phone == phone {
			return nil, ErrPhoneRegistered
		}	
	}
	s.NextAccountID++
	account := &types.Account {
		ID : s.NextAccountID,
		Phone: phone,
		Balance: 0,
	}
	s.accounts = append(s.accounts, account)

	return account, nil
}


// Deposit пополняет счёт пользователя
func (s *Service) Deposit(accountID int64, amount types.Money) error {
	if amount <= 0 {
		return ErrAmountMustBePositive
	}

	var account *types.Account
	for _, acc := range s.accounts {
		if acc.ID  == accountID {
			account = acc
			break
		}
	}

	if account == nil {
		return ErrAccountNotFound
	}

	account.Balance += amount
	return nil
}


// Pay платит определенную сумму денег за категорию
func (s *Service) Pay(accountID int64, amount types.Money, category types.PaymentCategory) (*types.Payment, error) {
	if amount <= 0 {
		return nil, ErrAmountMustBePositive
	}

	var account *types.Account
	for _, acc := range s.accounts {
		if acc.ID  == accountID {
			account = acc
			break
		}
	}
	if account == nil {
		return nil, ErrAccountNotFound
	}
	if account.Balance < amount {
		return nil, ErrNotEnoughtBalance
	}

	account.Balance -= amount
	paymentID := uuid.New().String()
	payment := &types.Payment {
		ID: paymentID,
		AccountID: accountID,
		Amount: amount,
		Category: category,
		Status: types.PaymentStatusInProgress,
	}
	s.payments = append(s.payments, payment)
	return payment, nil
}


// FindAccountById ищет пользователя по ID
func (s *Service) FindAccountByID(accountID int64) (*types.Account, error) {
	var account *types.Account
	for _, acc := range s.accounts {
		if acc.ID == accountID {
			account = acc
			break
		}
	}

	if account == nil {
		return nil, ErrAccountNotFound
	}

	return account, nil
}


// FindPaymentByID ищет платёж по ID
func (s *Service) FindPaymentByID(paymentID string) (*types.Payment, error) {
	var payment *types.Payment

	for _, pay := range s.payments {
		if pay.ID == paymentID {
			payment = pay
		}
	}

	if payment == nil {
		return nil, ErrPaymentNotFound
	}

	return payment, nil
}


// Reject отменяет платёж
func (s *Service) Reject(paymentID string) error {
	pay, err := s.FindPaymentByID(paymentID)
	if err != nil {
		return ErrPaymentNotFound
	}

	acc, err := s.FindAccountByID(pay.AccountID)
	if err != nil {
		return ErrAccountNotFound
	}

	pay.Status = types.PaymentStatusFail
	acc.Balance += pay.Amount

	return nil
}


// Repeat повторяет платёж по идентификатору 
func (s *Service) Repeat(paymentID string) (*types.Payment, error) {
	pay, err := s.FindPaymentByID(paymentID)
	if err != nil {
	  return nil, err
	}
  
	payment, err := s.Pay(pay.AccountID, pay.Amount, pay.Category)
	if err != nil {
	  return nil, err
	}
  
	return payment, nil
}

// FindFavoriteByID ищет платёж по ID в Избранное
func (s *Service) FindFavoriteByID(favoriteID string) (*types.Favorite, error) {
	for _, favorite := range s.favorites {
		if favorite.ID == favoriteID {
			return favorite, nil
		}
	}
	return nil, ErrFavoriteNotFound
}

// FavoritePayment добавляет новый платеж в Избранное
func (s *Service) FavoritePayment(paymentID string, name string) (*types.Favorite, error) {
	payment, err := s.FindPaymentByID(paymentID)

	if err != nil {
		return nil, err
	}

	favoriteID := uuid.New().String()
	newFavorite := &types.Favorite{
		ID:        favoriteID,
		AccountID: payment.AccountID,
		Name:      name,
		Amount:    payment.Amount,
		Category:  payment.Category,
	}

	s.favorites = append(s.favorites, newFavorite)
	return newFavorite, nil
}

// PayFromFavorite позволяет совершить платеж из Избранное
func (s *Service) PayFromFavorite(favoriteID string) (*types.Payment, error) {
	favorite, err := s.FindFavoriteByID(favoriteID)
	if err != nil {
		return nil, err
	}

	payment, err := s.Pay(favorite.AccountID, favorite.Amount, favorite.Category)
	if err != nil {
		return nil, err
	}

	return payment, nil
}

// ExportToFile экспортирует все аккаунты в файл, путь к которому указан в переменной path
func (s *Service) ExportToFile(path string) error {
	file, err := os.Create(path)

	if err != nil {
		log.Print(err)
		return ErrFileNotFound
	}

	defer func() {
		err := file.Close(); 
		if err != nil {
			log.Print(err)
		}
	}()

	stringAccount := ""

	for _, acc := range s.accounts {
		ID := strconv.Itoa(int(acc.ID))
		phone := string(acc.Phone)
		balance := strconv.Itoa(int(acc.Balance))

		stringAccount += ID + ";"
		stringAccount += phone + ";"
		stringAccount += balance + "|"
	}

	_, err = file.Write([]byte(stringAccount))
	if err != nil {
		log.Print(err)
		return ErrFileNotFound
	}

	return nil
}

//Export method
func (s *Service) Export(dir string) error {
	if len(s.accounts) > 0 {
		file, err := os.OpenFile(dir+"/accounts.dump", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)

		defer func() {
			if cerr := file.Close(); cerr != nil {
				if err != nil {
					err = cerr
					log.Print(err)
				}
			}
		}()

		str := ""
		for _, v := range s.accounts {
			str += fmt.Sprint(v.ID) + ";" + string(v.Phone) + ";" + fmt.Sprint(v.Balance) + "\n"
		}
		file.WriteString(str)
	}
	if len(s.payments) > 0 {
		file, _ := os.OpenFile(dir+"/payments.dump", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)

		defer func() {
			if cerr := file.Close(); cerr != nil {
				if err != nil {
					err = cerr
					log.Print(err)
				}
			}
		}()

		str := ""
		for _, v := range s.payments {
			str += fmt.Sprint(v.ID) + ";" + fmt.Sprint(v.AccountID) + ";" + fmt.Sprint(v.Amount) + ";" + fmt.Sprint(v.Category) + ";" + fmt.Sprint(v.Status) + "\n"
		}
		file.WriteString(str)
	}

	if len(s.favorites) > 0 {
		file, _ := os.OpenFile(dir+"/favorites.dump", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)

		defer func() {
			if cerr := file.Close(); cerr != nil {
				if err != nil {
					err = cerr
					log.Print(err)
				}
			}
		}()

		str := ""
		for _, v := range s.favorites {
			str += fmt.Sprint(v.ID) + ";" + fmt.Sprint(v.AccountID) + ";" + fmt.Sprint(v.Amount) + ";" + fmt.Sprint(v.Category) + "\n"
		}
		file.WriteString(str)
	}

	return nil
}

// ImportFromFile импортирует все данные из файла, путь к которому указан в переменной path
func (s *Service) ImportFromFile(path string) error {
	file, err := os.Open(path)

	if err != nil {
		log.Print(err)
		return ErrFileNotFound
	}

	defer func() {
		err := file.Close(); 
		if err != nil {
			log.Print(err)
		}
	}()

	content := make([]byte, 0)
	buf := make([]byte, 4)
	for {
		read, err := file.Read(buf)
		if err == io.EOF {
			break
		}

		if err != nil {
			log.Print(err)
			return ErrFileNotFound
		}
		content = append(content, buf[:read]...)
	}

	data := string(content)

	accountsString := strings.Split(string(data), "|")

	if len(accountsString) > 0 {
		accountsString = accountsString[:len(accountsString)-1]
	}

	for _, acc := range accountsString {
		fields := strings.Split(acc, ";")

		ID, err := strconv.Atoi(fields[0])
		if err != nil {
			return err
		}

		phone := fields[1]

		balance, err := strconv.Atoi(fields[2])
		if err != nil {
			return err
		}

		account := &types.Account{
			ID:      int64(ID),
			Phone:   types.Phone(phone),
			Balance: types.Money(balance),
		}

		s.accounts = append(s.accounts, account)
	}

	return nil
}

//Import method
func (s *Service) Import(dir string) error {

	_, err := os.Stat(dir + "/accounts.dump")

	if err == nil {
		content, err := ioutil.ReadFile(dir + "/accounts.dump")
		if err != nil {
			return err
		}

		strArray := strings.Split(string(content), "\n")
		if len(strArray) > 0 {
			strArray = strArray[:len(strArray)-1]
		}
		for _, v := range strArray {
			strArrAcount := strings.Split(v, ";")
			fmt.Println(strArrAcount)

			id, err := strconv.ParseInt(strArrAcount[0], 10, 64)
			if err != nil {
				return err
			}
			balance, err := strconv.ParseInt(strArrAcount[2], 10, 64)
			if err != nil {
				return err
			}
			flag := true
			for _, v := range s.accounts {
				if v.ID == id {
					v.Phone = types.Phone(strArrAcount[1])
					v.Balance = types.Money(balance)
					flag = false
				}
			}
			if flag {
				account := &types.Account{
					ID:      id,
					Phone:   types.Phone(strArrAcount[1]),
					Balance: types.Money(balance),
				}
				s.accounts = append(s.accounts, account)
			}
		}
	}

	_, err1 := os.Stat(dir + "/payments.dump")

	if err1 == nil {
		content, err := ioutil.ReadFile(dir + "/payments.dump")
		if err != nil {
			return err
		}

		strArray := strings.Split(string(content), "\n")
		if len(strArray) > 0 {
			strArray = strArray[:len(strArray)-1]
		}
		for _, v := range strArray {
			strArrAcount := strings.Split(v, ";")
			fmt.Println(strArrAcount)

			id := strArrAcount[0]
			if err != nil {
				return err
			}
			aid, err := strconv.ParseInt(strArrAcount[1], 10, 64)
			if err != nil {
				return err
			}
			amount, err := strconv.ParseInt(strArrAcount[2], 10, 64)
			if err != nil {
				return err
			}
			flag := true
			for _, v := range s.payments {
				if v.ID == id {
					v.AccountID = aid
					v.Amount = types.Money(amount)
					v.Category = types.PaymentCategory(strArrAcount[3])
					v.Status = types.PaymentStatus(strArrAcount[4])
					flag = false
				}
			}
			if flag {
				data := &types.Payment{
					ID:        id,
					AccountID: aid,
					Amount:    types.Money(amount),
					Category:  types.PaymentCategory(strArrAcount[3]),
					Status:    types.PaymentStatus(strArrAcount[4]),
				}
				s.payments = append(s.payments, data)
			}
		}
	}

	_, err2 := os.Stat(dir + "/favorites.dump")

	if err2 == nil {
		content, err := ioutil.ReadFile(dir + "/favorites.dump")
		if err != nil {
			return err
		}

		strArray := strings.Split(string(content), "\n")
		if len(strArray) > 0 {
			strArray = strArray[:len(strArray)-1]
		}
		for _, v := range strArray {
			strArrAcount := strings.Split(v, ";")
			fmt.Println(strArrAcount)

			id := strArrAcount[0]
			if err != nil {
				return err
			}
			aid, err := strconv.ParseInt(strArrAcount[1], 10, 64)
			if err != nil {
				return err
			}
			amount, err := strconv.ParseInt(strArrAcount[2], 10, 64)
			if err != nil {
				return err
			}
			flag := true
			for _, v := range s.favorites {
				if v.ID == id {
					v.AccountID = aid
					v.Amount = types.Money(amount)
					v.Category = types.PaymentCategory(strArrAcount[3])
					flag = false
				}
			}
			if flag {
				data := &types.Favorite{
					ID:        id,
					AccountID: aid,
					Amount:    types.Money(amount),
					Category:  types.PaymentCategory(strArrAcount[3]),
				}
				s.favorites = append(s.favorites, data)
			}
		}
	}

	return nil
}

//ExportAccountHistory вытаскивает все платежи конкретного аккаунта, если их нет - возвращает ошибку
func (s *Service) ExportAccountHistory(accountID int64) ([]types.Payment, error) {

	account, err := s.FindAccountByID(accountID)

	if err != nil {
		return nil, err
	}

	var payments []types.Payment
	for _, v := range s.payments {
		if v.AccountID == account.ID {
			data := types.Payment{
				ID:        v.ID,
				AccountID: v.AccountID,
				Amount:    v.Amount,
				Category:  v.Category,
				Status:    v.Status,
			}
			payments = append(payments, data)
		}
	}
	return payments, nil
}

//HistoryToFiles сохраняет данные из предыдущего метода
func (s *Service) HistoryToFiles(payments []types.Payment, dir string, records int) error {

	if len(payments) > 0 {
		if len(payments) <= records {
			file, _ := os.OpenFile(dir+"/payments.dump", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
			defer file.Close()

			var str string
			for _, v := range payments {
				str += fmt.Sprint(v.ID) + ";" + fmt.Sprint(v.AccountID) + ";" + fmt.Sprint(v.Amount) + ";" + fmt.Sprint(v.Category) + ";" + fmt.Sprint(v.Status) + "\n"
			}
			file.WriteString(str)
		} else {

			var str string
			k := 0
			t := 1
			var file *os.File
			for _, v := range payments {
				if k == 0{
				file, _ = os.OpenFile(dir+"/payments"+fmt.Sprint(t)+".dump", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
				}
				k ++
				str = fmt.Sprint(v.ID) + ";" + fmt.Sprint(v.AccountID) + ";" + fmt.Sprint(v.Amount) + ";" + fmt.Sprint(v.Category) + ";" + fmt.Sprint(v.Status) + "\n"
				_, err = file.WriteString(str)
				if k == records{
					str = ""
					t ++
					k = 0;
					file.Close()
				}
			}

		}
	}

	return nil
}

//SumPayments суммирует платежи
func (s *Service) SumPayments(goroutines int) types.Money {
	wg := sync.WaitGroup{}
	mu := sync.Mutex{}
	sum := int64(0)
	kol := 0
	i := 0
	if goroutines == 0 {
		kol = len(s.payments)
	} else {
		kol = int(len(s.payments) / goroutines)
	}
	for i = 0; i < goroutines-1; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			val := int64(0)
			payments := s.payments[index*kol : (index+1)*kol]
			for _, payment := range payments {
				val += int64(payment.Amount)
			}
			mu.Lock()
			sum += val
			mu.Unlock()

		}(i)
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		val := int64(0)
		payments := s.payments[i*kol:]
		for _, payment := range payments {
			val += int64(payment.Amount)
		}
		mu.Lock()
		sum += val
		mu.Unlock()

	}()
	wg.Wait()
	return types.Money(sum)
}

//FilterPayments отфильтровывает платежи, выдавая нам только те, у которых accountID равен переданному
func (s *Service) FilterPayments(accountID int64, goroutines int) ([]types.Payment, error) {

	account, err := s.FindAccountByID(accountID)

	if err != nil {
		return nil, err
	}

	wg := sync.WaitGroup{}
	mu := sync.Mutex{}
	kol := 0
	i := 0
	var ps []types.Payment
	if goroutines == 0 {
		kol = len(s.payments)
	} else {
		kol = int(len(s.payments) / goroutines)
	}
	for i = 0; i < goroutines-1; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			var pays []types.Payment
			payments := s.payments[index*kol : (index+1)*kol]
			for _, v := range payments {
				if v.AccountID == account.ID {
					pays = append(pays, types.Payment{
						ID:        v.ID,
						AccountID: v.AccountID,
						Amount:    v.Amount,
						Category:  v.Category,
						Status:    v.Status,
					})
				}
			}
			mu.Lock()
			ps = append(ps, pays...)
			mu.Unlock()

		}(i)
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		var pays []types.Payment
		payments := s.payments[i*kol:]
		for _, v := range payments {
			if v.AccountID == account.ID {
				pays = append(pays, types.Payment{
					ID:        v.ID,
					AccountID: v.AccountID,
					Amount:    v.Amount,
					Category:  v.Category,
					Status:    v.Status,
				})
			}
		}
		mu.Lock()
		ps = append(ps, pays...)
		mu.Unlock()

	}()
	wg.Wait()
	if len(ps)==0{
		return nil, nil
	}
	return  ps, nil
}

//FilterPaymentsByFn отфильтровывает платежи, выдавая только те где filter(payment) == true
func (s *Service) FilterPaymentsByFn(filter func(payment types.Payment) bool, goroutines int,) ([]types.Payment, error){

	wg := sync.WaitGroup{}
	mu := sync.Mutex{}
	kol := 0
	i := 0
	var ps []types.Payment
	if goroutines == 0 {
		kol = len(s.payments)
	} else {
		kol = int(len(s.payments) / goroutines)
	}
	for i = 0; i < goroutines-1; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			var pays []types.Payment
			payments := s.payments[index*kol : (index+1)*kol]
			for _, v := range payments {
				p := types.Payment{
					ID:        v.ID,
					AccountID: v.AccountID,
					Amount:    v.Amount,
					Category:  v.Category,
					Status:    v.Status,
				}

				if filter(p) {
					pays = append(pays, p)
				}
			}
			mu.Lock()
			ps = append(ps, pays...)
			mu.Unlock()

		}(i)
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		var pays []types.Payment
		payments := s.payments[i*kol:]
		for _, v := range payments {

			p := types.Payment{
				ID:        v.ID,
				AccountID: v.AccountID,
				Amount:    v.Amount,
				Category:  v.Category,
				Status:    v.Status,
			}

			if filter(p) {
				pays = append(pays, p)
			}
		}
		mu.Lock()
		ps = append(ps, pays...)
		mu.Unlock()

	}()
	wg.Wait()
	if len(ps) == 0{
		return nil, nil
	}
	return  ps, nil
}
//SumPaymentsWithProgress делит платежи на куски по 100_000 платежей в каждом и суммирует их параллельно друг другу
func (s *Service) SumPaymentsWithProgress() <-chan types.Progress {
	sizeOfUnit := 100_0000 		/* когда условие и требование в задаче не совпадают :) */

	wg := sync.WaitGroup{}
	goroutines := len(s.payments) / sizeOfUnit /* определяем количество горутин - сколько кусков потребуется сложить*/
	if goroutines <= 1 {
		goroutines = 1	
	/* на случай если платеж всего один (или их нет) */
	}
	ch := make(chan types.Progress)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(ch chan <- types.Progress, payments []*types.Payment) {
			//defer close(ch)
			var sum types.Money = 0
			defer wg.Done()
			for _, pay := range payments {
				sum += pay.Amount
			}
			ch <- types.Progress{
				Part:   len(payments), 
				Result: sum,
			}
		}(ch, s.payments)
	}

	go func() {
		defer close(ch)
		wg.Wait()
	}()

	return ch
} 