package main

import (
	"database/sql"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

var (
	// randSource источник псевдо случайных чисел.
	// Для повышения уникальности в качестве seed
	// используется текущее время в unix формате (в виде числа)
	randSource = rand.NewSource(time.Now().UnixNano())
	// randRange использует randSource для генерации случайных чисел
	randRange = rand.New(randSource)
)

// openTestDB открывает соединение с тестовой базой данных и возвращает его.
func openTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", "tracker.db")
	require.NoError(t, err)
	require.NoError(t, db.Ping())

	t.Cleanup(func() {
		_ = db.Close()
	})

	return db
}

func cleanupParcel(t *testing.T, db *sql.DB, number int) {
	t.Helper()
	_, err := db.Exec("DELETE FROM parcel WHERE number = ?", number)
	require.NoError(t, err)
}

func getTestParcel() Parcel {
	return Parcel{
		Client:    1000,
		Status:    ParcelStatusRegistered,
		Address:   "test",
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
}

// TestAddGetDelete проверяет добавление, получение и удаление посылки
func TestAddGetDelete(t *testing.T) {
	db := openTestDB(t) // настройте подключение к БД
	store := NewParcelStore(db)
	parcel := getTestParcel()

	id, err := store.Add(parcel)
	require.NoError(t, err)
	require.NotZero(t, id)
	t.Cleanup(func() { cleanupParcel(t, db, id) })

	parcel.Number = id

	storedParcel, err := store.Get(id)
	require.NoError(t, err)
	require.Equal(t, parcel.Number, storedParcel.Number)
	require.Equal(t, parcel.Client, storedParcel.Client)
	require.Equal(t, parcel.Status, storedParcel.Status)
	require.Equal(t, parcel.Address, storedParcel.Address)
	require.Equal(t, parcel.CreatedAt, storedParcel.CreatedAt)

	err = store.Delete(id)
	require.NoError(t, err)

	_, err = store.Get(id)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

// TestSetAddress проверяет обновление адреса
func TestSetAddress(t *testing.T) {
	db := openTestDB(t) // настройте подключение к БД
	store := NewParcelStore(db)
	parcel := getTestParcel()

	id, err := store.Add(parcel)
	require.NoError(t, err)
	require.NotZero(t, id)
	t.Cleanup(func() { cleanupParcel(t, db, id) })

	newAddress := "new test address"
	err = store.SetAddress(id, newAddress)
	require.NoError(t, err)

	storedParcel, err := store.Get(id)
	require.NoError(t, err)
	require.Equal(t, newAddress, storedParcel.Address)
}

// TestSetStatus checks status update.
func TestSetStatus(t *testing.T) {
	db := openTestDB(t) // настройте подключение к БД
	store := NewParcelStore(db)
	parcel := getTestParcel()

	id, err := store.Add(parcel)
	require.NoError(t, err)
	require.NotZero(t, id)
	t.Cleanup(func() { cleanupParcel(t, db, id) })

	err = store.SetStatus(id, ParcelStatusSent)
	require.NoError(t, err)

	storedParcel, err := store.Get(id)
	require.NoError(t, err)
	require.Equal(t, ParcelStatusSent, storedParcel.Status)
}

// TestGetByClient проверяет получение посылок по идентификатору клиента
func TestGetByClient(t *testing.T) {
	db := openTestDB(t) // настройте подключение к БД
	store := NewParcelStore(db)

	parcels := []Parcel{
		getTestParcel(),
		getTestParcel(),
		getTestParcel(),
	}
	parcelMap := map[int]Parcel{}

	// задаём всем посылкам один и тот же идентификатор клиента
	client := randRange.Intn(10_000_000)
	for i := 0; i < len(parcels); i++ {
		parcels[i].Client = client
	}

	for i := 0; i < len(parcels); i++ {
		id, err := store.Add(parcels[i]) // добавьте новую посылку в БД, убедитесь в отсутствии ошибки и наличии идентификатора
		require.NoError(t, err)
		require.NotZero(t, id)
		// обновляем идентификатор добавленной у посылки
		t.Cleanup(func(parcelID int) func() {
			return func() { cleanupParcel(t, db, parcelID) }
		}(id))

		parcels[i].Number = id
		// сохраняем добавленную посылку в структуру map, чтобы её можно было легко достать по идентификатору посылки
		parcelMap[id] = parcels[i]
	}

	storedParcels, err := store.GetByClient(client) // получите список посылок по идентификатору клиента, сохранённого в переменной client
	require.NoError(t, err)
	require.Len(t, storedParcels, len(parcels))

	for _, parcel := range storedParcels {
		// в parcelMap лежат добавленные посылки, ключ - идентификатор посылки, значение - сама посылка
		// убедитесь, что все посылки из storedParcels есть в parcelMap
		// убедитесь, что значения полей полученных посылок заполнены верно
		expectedParcel, ok := parcelMap[parcel.Number]
		require.True(t, ok)
		require.Equal(t, expectedParcel.Number, parcel.Number)
		require.Equal(t, expectedParcel.Client, parcel.Client)
		require.Equal(t, expectedParcel.Status, parcel.Status)
		require.Equal(t, expectedParcel.Address, parcel.Address)
		require.Equal(t, expectedParcel.CreatedAt, parcel.CreatedAt)
	}
}
