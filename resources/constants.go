package resources

// The password for testing keystore files:
//
// 	./test_andrej--3eb92807f1f91a8d4d85bc908c7f86dcddb1df57
// 	./test_babayaga--6fdc0d8d15ae6b4ebf45c52fd2aafbcbb19a65c8
//
// Pre-generated for testing purposes using wallet_test.go.
//
// It's necessary to have pre-existing accounts before a new node
// with fresh new, empty keystore is initialized and booted in order
// to configure the accounts balances in genesis.json
//
// I.e: A quick solution to a chicken-egg problem.

const TestKsAndrejAccount = "0x3eb92807f1f91a8d4d85bc908c7f86dcddb1df57"
const TestKsBabaYagaAccount = "0x6fdc0d8d15ae6b4ebf45c52fd2aafbcbb19a65c8"
const TestKsAndrejFile = "test_andrej--3eb92807f1f91a8d4d85bc908c7f86dcddb1df57"
const TestKsBabaYagaFile = "test_babayaga--6fdc0d8d15ae6b4ebf45c52fd2aafbcbb19a65c8"
const TestKsAccountsPwd = "security123"
