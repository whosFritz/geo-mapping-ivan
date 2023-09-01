package main

import "testing"

func TestReadLastLine(t *testing.T) {
	testCases := []struct {
		input        string
		expectedUser string
		expectedIP   string
	}{
		{
			input:        "Sep 1 08:54:46 localhost sshd[6061]: Failed password for invalid user blank from 183.83.218.176 port 44794 ssh2",
			expectedUser: "blank",
			expectedIP:   "183.83.218.176",
		},
		{
			input:        "Sep  1 08:54:27 localhost sshd[6057]: Failed password for root from 31.41.244.62 port 53260 ssh2",
			expectedUser: "root",
			expectedIP:   "31.41.244.62",
			// Add more test cases here if needed
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.input, func(t *testing.T) {
			user, ip := extractData(testCase.input)
			if user != testCase.expectedUser || ip != testCase.expectedIP {
				t.Errorf("Expected user: %s, IP: %s, but got user: %s, IP: %s", testCase.expectedUser, testCase.expectedIP, user, ip)
			}
		})
	}
}
