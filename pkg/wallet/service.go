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

//ErrPhoneRegistered - телефон уже регитрирован
var ErrPhoneRegistered = errors.New("phone already registred")

//ErrAmountMustBePositive - счёт не может быть пустым
var ErrAmountMustBePositive = errors.New("amount must be greater than zero")

//ErrAccountNotFound - пользователь не найден
var ErrAccountNotFound = errors.New("account not found")

//ErrNotEnoughtBalance - на счете недостаточно средств
var ErrNotEnoughtBalance = errors.New("account not enough balance")

//ErrPaymentNotFound - платеж не найден
var ErrPaymentNotFound = errors.New("payment not found")

// ErrFavoriteNotFound - Избранное не найдено
var ErrFavoriteNotFound = errors.New("favorite not found")

// ErrFileNotFound - файл не найден
var ErrFileNotFound = errors.New("file not fount")

var err error

// Service представляет информацию о пользователе.
type Service struct {
	nextAccountID int64
	accounts      []*types.Account
	payments      []*types.Payment
	favorites     []*types.Favorite
}

// RegisterAccount - метод для регистрация нового прользователя.
func (s *Service) RegisterAccount(phone types.Phone) (*types.Account, error) {
	for _, account := range s.accounts {
		if account.Phone == phone {
			return nil, ErrPhoneRegistered
		}
	}
	s.nextAccountID++
	account := &types.Account{
		ID:      s.nextAccountID,
		Phone:   phone,
		Balance: 0,
	}
	s.accounts = append(s.accounts, account)
	return account, nil
}

//Pay метод для регистрации платижа
func (s *Service) Pay(accountID int64, amount types.Money, category types.PaymentCategory) (*types.Payment, error) {
	if amount <= 0 {
		return nil, ErrAmountMustBePositive
	}

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

	if account.Balance < amount {
		return nil, ErrNotEnoughtBalance
	}

	account.Balance -= amount

	paymentID := uuid.New().String()
	payment := &types.Payment{
		ID:        paymentID,
		AccountID: accountID,
		Amount:    amount,
		Category:  category,
		Status:    types.PaymentStatusInProgress,
	}

	s.payments = append(s.payments, payment)
	return payment, nil
}

// FindAccountByID ищем пользователя по ID
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

// FindPaymentByID ищем платёж по ID
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

// FindFavoriteByID ищем платёж по ID в Избранное
func (s *Service) FindFavoriteByID(favoriteID string) (*types.Favorite, error) {
	for _, favorite := range s.favorites {
		if favorite.ID == favoriteID {
			return favorite, nil
		}
	}
	return nil, ErrFavoriteNotFound
}

//Deposit method
func (s *Service) Deposit(accountID int64, amount types.Money) error {
	if amount < 0 {
		return ErrAmountMustBePositive
	}

	account, err := s.FindAccountByID(accountID)
	if err != nil {
		return err
	}

	account.Balance += amount
	return nil
}

// Reject метод для отмены покупок
func (s *Service) Reject(paymentID string) error {
	pay, err := s.FindPaymentByID(paymentID)
	if err != nil {
		return err
	}

	acc, err := s.FindAccountByID(pay.AccountID)
	if err != nil {
		return err
	}

	pay.Status = types.PaymentStatusFail
	acc.Balance += pay.Amount

	return nil
}

// Repeat позволāет по идентификатору повторитþ платёж
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

// FavoritePayment добавления новых Избранных
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

//PayFromFavorite для совершения платежа в Избранное
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

// ExportToFile экспартирует данные аккаунтов в файл
func (s *Service) ExportToFile(path string) error {
	file, err := os.Create(path)
	if err != nil {
		log.Print(err)
		return ErrFileNotFound
	}
	defer func() {
		if cerr := file.Close(); cerr != nil {
			log.Print(cerr)
		}
	}()

	str := ""

	for _, acc := range s.accounts {
		ID := strconv.Itoa(int(acc.ID)) + ";"
		phone := string(acc.Phone) + ";"
		balance := strconv.Itoa(int(acc.Balance))

		str += ID
		str += phone
		str += balance + "|"
	}
	_, err = file.Write([]byte(str))
	if err != nil {
		log.Print(err)
		return ErrFileNotFound
	}

	return nil
}

// ImportFromFile импортирует данные с аккаунта
func (s *Service) ImportFromFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		log.Print(err)
		return ErrFileNotFound
	}
	defer func() {
		err := file.Close()
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

	accounts := strings.Split(string(data), "|")
	if len(accounts) > 0 {
		accounts = accounts[:len(accounts)-1] // это фигня служит для того чтоб цикл пощел на n - 1, т.к split добавил в слайс ещё один элемент (причём пустой)
	}
	for _, acc := range accounts {
		splits := strings.Split(acc, ";")

		id, err := strconv.Atoi(splits[0])
		if err != nil {
			return err
		}

		balance, err := strconv.Atoi(splits[2])
		if err != nil {
			return err
		}

		newAccount := &types.Account{
			ID:      int64(id),
			Phone:   types.Phone(splits[1]),
			Balance: types.Money(balance),
		}

		s.accounts = append(s.accounts, newAccount)
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
		file, err := os.OpenFile(dir+"/payments.dump", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)

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
		file, err := os.OpenFile(dir+"/favorites.dump", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)

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

// ExportAccountHistory ...
func (s *Service) ExportAccountHistory(accountID int64) ([]types.Payment, error) {
	acc, err := s.FindAccountByID(accountID)
	if err != nil {
		return nil, err
	}

	var payments []types.Payment

	for _, pay := range s.payments {
		if pay.AccountID == acc.ID {
			data := types.Payment{
				ID:        pay.ID,
				AccountID: pay.AccountID,
				Amount:    pay.Amount,
				Category:  pay.Category,
				Status:    pay.Status,
			}
			payments = append(payments, data)
		}
	}
	return payments, nil
}

//HistoryToFiles ...
func (s *Service) HistoryToFiles(payments []types.Payment, dir string, records int) error {

	str := ""

	if len(payments) > 0 && len(payments) <= records {
		file, _ := os.OpenFile(dir+"/payments.dump", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
		defer file.Close()

		for _, v := range payments {
			str += fmt.Sprint(v.ID) + ";" + fmt.Sprint(v.AccountID) + ";" + fmt.Sprint(v.Amount) + ";" + fmt.Sprint(v.Category) + ";" + fmt.Sprint(v.Status) + "\n"
		}
		file.WriteString(str)
	} else {
		k := 0 // limit on record
		t := 1 // count for files
		var file *os.File
		for _, v := range payments {
			if k == 0 {
				file, _ = os.OpenFile(dir+"/payments"+fmt.Sprint(t)+".dump", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
			}
			k++
			str = fmt.Sprint(v.ID) + ";" + fmt.Sprint(v.AccountID) + ";" + fmt.Sprint(v.Amount) + ";" + fmt.Sprint(v.Category) + ";" + fmt.Sprint(v.Status) + "\n"
			_, err = file.WriteString(str)
			if k == records { // если лимит был дастигнут, то обнулить "записи"
				str = ""
				t++
				k = 0
				file.Close()
			}
		}
	}
	return nil
}

//SumPayments ...
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

//FilterPayments ...
func (s *Service) FilterPayments(accountID int64, goroutines int) ([]types.Payment, error) {

	account, err := s.FindAccountByID(accountID)

	if err != nil {
		return nil, err
	}

	wg := sync.WaitGroup{}
	mu := sync.Mutex{}
	kol := 0
	i := 0
	ps := []types.Payment{}
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
			ps = pays[:]
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
		ps = pays
		mu.Unlock()

	}()
	wg.Wait()
	if len(ps) == 0 {
		return nil, nil
	}
	return ps, nil
}

//FilterPaymentsByFn ...
func (s *Service) FilterPaymentsByFn(filter func(payment types.Payment) bool, goroutines int) ([]types.Payment, error) {

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
			ps = pays[:]
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
		ps = pays[:]
		mu.Unlock()

	}()
	wg.Wait()
	if len(ps) == 0 {
		return nil, nil
	}
	return ps, nil
}

// SumPaymentsWithProgress суммируем и делим все платежи по кусочком
func (s *Service) SumPaymentsWithProgress() <-chan types.Progress {
	size := 100_0000

	amountOfMoney := []types.Money{}
	for _, pay := range s.payments {
		amountOfMoney = append(amountOfMoney, pay.Amount)
	}

	wg := sync.WaitGroup{}
	goroutines := (len(amountOfMoney) + 1) / size
	if goroutines <= 0 {
		goroutines = 1
	}
	ch := make(chan types.Progress)
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(ch chan<- types.Progress, amountOfMoney []types.Money) {
			sum := 0
			defer wg.Done()
			for _, val := range amountOfMoney {
				sum += int(val)

			}
			ch <- types.Progress{
				Part:   len(amountOfMoney),
				Result: types.Money(sum),
			}
		}(ch, amountOfMoney)
	}

	go func() {
		defer close(ch)
		wg.Wait()
	}()

	return ch
}