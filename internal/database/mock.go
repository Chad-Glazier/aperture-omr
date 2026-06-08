package database

//
// In this file, we implement a mock database. This version should not rely
// on any external connections, instead storing objects directly in memory.
//

type mockData struct{}

var data = mockData{}

func GetMockDB() *DB {
	panic("MockDB not implemented yet.")
}
