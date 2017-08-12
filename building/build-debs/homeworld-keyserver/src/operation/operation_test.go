package operation

import (
	"testing"
	"account"
	"config"
	"errors"
	"fmt"
	"log"
	"bytes"
	"strings"
)

func TestInvokeAPIOperation(t *testing.T) {
	opctx := account.OperationContext{}
	gctx := config.Context{
		Grants: map[string]config.Grant {
			"test-api": {
				API: "test-api",
				PrivilegeByAccount: map[string]account.Privilege {
					"test-account": func(iopctx *account.OperationContext, param string) (string, error) {
						if iopctx != &opctx {
							return "", errors.New("Mismatched opctx.")
						}
						return fmt.Sprintf("cheap plastic %s, made in china", param), nil
					},
					"test-account-2": func(iopctx *account.OperationContext, param string) (string, error) {
						if iopctx != &opctx {
							return "", errors.New("Mismatched opctx.")
						}
						return "", errors.New("A testing error.")
					},
				},
			},
		},
	}
	buf := bytes.NewBuffer(nil)
	logger := log.New(buf, "", 0)
	opctx.Account = &account.Account{Principal:"test-account"}
	result, err := InvokeAPIOperation(&opctx, &gctx, "test-api", "gemstone", logger)
	if err != nil {
		t.Error(err)
	} else if result != "cheap plastic gemstone, made in china" {
		t.Error("Wrong result.")
	}
	opctx.Account = &account.Account{Principal:"test-account-2"}
	_, err = InvokeAPIOperation(&opctx, &gctx, "test-api", "gemstone", logger)
	if err == nil {
		t.Error("Expected error.")
	} else if err.Error() != "A testing error." {
		t.Error("Wrong error.")
	}
	lines := []string { "Attempting to perform API operation test-api for test-account",
                        "Operation test-api for test-account succeeded.",
						"Attempting to perform API operation test-api for test-account-2",
						"Operation test-api for test-account-2 failed with error: A testing error.",
						""}
	found := strings.Split(buf.String(), "\n")
	if len(lines) != len(found) {
		t.Error("Wrong number of log lines.")
	} else {
		for i, expect := range lines {
			if found[i] != expect {
				t.Error("Log line mismatch.")
			}
		}
	}
}

func TestInvokeAPIOperation_NoAPI(t *testing.T) {
	opctx := account.OperationContext{Account: &account.Account{Principal:"test-account"}}
	gctx := config.Context{}
	buf := bytes.NewBuffer(nil)
	logger := log.New(buf, "", 0)
	_, err := InvokeAPIOperation(&opctx, &gctx, "test-api", "gemstone", logger)
	if err == nil {
		t.Error("Expected error.")
	} else if !strings.Contains(err.Error(), "Could not find API request") {
		t.Error("Wrong error.")
	}
	if buf.String() != "" {
		t.Error("Expected no logging.")
	}
}

func TestInvokeAPIOperation_NoAccount(t *testing.T) {
	opctx := account.OperationContext{}
	gctx := config.Context{
		Grants: map[string]config.Grant {
			"test-api": {
				API: "test-api",
			},
		},
	}
	buf := bytes.NewBuffer(nil)
	logger := log.New(buf, "", 0)
	_, err := InvokeAPIOperation(&opctx, &gctx, "test-api", "gemstone", logger)
	if err == nil {
		t.Error("Expected error.")
	} else if !strings.Contains(err.Error(), "Missing account") {
		t.Error("Wrong error.")
	}
	if buf.String() != "" {
		t.Error("Expected no logging.")
	}
}

func TestInvokeAPIOperation_NoAccess(t *testing.T) {
	opctx := account.OperationContext{Account: &account.Account{Principal:"test-account"}}
	gctx := config.Context{
		Grants: map[string]config.Grant {
			"test-api": {
				API: "test-api",
			},
		},
	}
	buf := bytes.NewBuffer(nil)
	logger := log.New(buf, "", 0)
	_, err := InvokeAPIOperation(&opctx, &gctx, "test-api", "gemstone", logger)
	if err == nil {
		t.Error("Expected error.")
	} else if !strings.Contains(err.Error(), "does not have access") {
		t.Error("Wrong error.")
	}
	if buf.String() != "" {
		t.Error("Expected no logging.")
	}
}

func TestInvokeAPIOperationSet(t *testing.T) {
	gctx := config.Context{
		Grants: map[string]config.Grant {
			"test-api": {
				API: "test-api",
				PrivilegeByAccount: map[string]account.Privilege {
					"test-account": func(iopctx *account.OperationContext, param string) (string, error) {
						return fmt.Sprintf("cheap 3d-printed %s, made in our basement", param), nil
					},
				},
			},
			"test-api-2": {
				API: "test-api-2",
				PrivilegeByAccount: map[string]account.Privilege {
					"test-account": func(iopctx *account.OperationContext, param string) (string, error) {
						return fmt.Sprintf("cheap plastic %s, made in china", param), nil
					},
				},
			},
		},
	}
	test_account := &account.Account{Principal: "test-account"}
	buf := bytes.NewBuffer(nil)
	logger := log.New(buf, "", 0)
	sample := "[{\"api\": \"test-api-2\", \"body\": \"arm\"}, {\"api\": \"test-api\", \"body\": \"leg\"}]"
	result, err := InvokeAPIOperationSet(test_account, &gctx, []byte(sample), logger)
	if err != nil {
		t.Error(err)
	} else if string(result) != "[\"cheap plastic arm, made in china\",\"cheap 3d-printed leg, made in our basement\"]" {
		t.Errorf("Wrong result %s", string(result))
	}
	lines := []string { "Attempting to perform API operation test-api-2 for test-account",
						"Operation test-api-2 for test-account succeeded.",
						"Attempting to perform API operation test-api for test-account",
						"Operation test-api for test-account succeeded.",
						""}
	found := strings.Split(buf.String(), "\n")
	if len(lines) != len(found) {
		t.Error("Wrong number of log lines.")
	} else {
		for i, expect := range lines {
			if found[i] != expect {
				t.Errorf("Log line mismatch '%s' instead of '%s'.", found[i], expect)
			}
		}
	}
}

func TestInvokeAPIOperationSet_Delegate(t *testing.T) {
	impersonate, err := account.NewImpersonatePrivilege(func(name string) (*account.Account, error) {
		if name == "test-account-2" {
			return &account.Account{Principal: "test-account-2"}, nil
		}
		return nil, fmt.Errorf("No such account %s", name)
	}, &account.Group{Name: "test-group", AllMembers: []string { "test-account-2" }})
	if err != nil {
		t.Fatal(err)
	}
	gctx := config.Context{
		Grants: map[string]config.Grant {
			"test-api": {
				API: "test-api",
				PrivilegeByAccount: map[string]account.Privilege {
					"test-account": impersonate,
				},
			},
			"test-api-2": {
				API: "test-api-2",
				PrivilegeByAccount: map[string]account.Privilege {
					"test-account-2": func(iopctx *account.OperationContext, param string) (string, error) {
						return fmt.Sprintf("cheap 3d-printed %s, made in our basement", param), nil
					},
				},
			},
		},
	}
	test_account := &account.Account{Principal: "test-account"}
	buf := bytes.NewBuffer(nil)
	logger := log.New(buf, "", 0)
	sample := "[{\"api\": \"test-api\", \"body\": \"test-account-2\"}, {\"api\": \"test-api-2\", \"body\": \"head\"}]"
	result, err := InvokeAPIOperationSet(test_account, &gctx, []byte(sample), logger)
	if err != nil {
		t.Error(err)
	} else if string(result) != "[\"\",\"cheap 3d-printed head, made in our basement\"]" {
		t.Errorf("Wrong result %s", string(result))
	}
	lines := []string { "Attempting to perform API operation test-api for test-account",
						"Operation test-api for test-account succeeded.",
						"Attempting to perform API operation test-api-2 for test-account-2",
						"Operation test-api-2 for test-account-2 succeeded.",
						""}
	found := strings.Split(buf.String(), "\n")
	if len(lines) != len(found) {
		t.Error("Wrong number of log lines.")
	} else {
		for i, expect := range lines {
			if found[i] != expect {
				t.Errorf("Log line mismatch '%s' instead of '%s'.", found[i], expect)
			}
		}
	}
}

func TestInvokeAPIOperationSet_FailJson(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	logger := log.New(buf, "", 0)
	_, err := InvokeAPIOperationSet(nil, nil, []byte("10"), logger)
	if err == nil {
		t.Error("Expected error.")
	} else if !strings.Contains(err.Error(), "cannot unmarshal") {
		t.Errorf("Wrong error: %s", err)
	}
	if buf.String() != "" {
		t.Error("Expected no logging.")
	}
}

func TestInvokeAPIOperationSet_FailAPI(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	logger := log.New(buf, "", 0)
	_, err := InvokeAPIOperationSet(nil, nil, []byte("[{}]"), logger)
	if err == nil {
		t.Error("Expected error.")
	} else if !strings.Contains(err.Error(), "Missing API") {
		t.Errorf("Wrong error: %s", err)
	}
	if buf.String() != "" {
		t.Error("Expected no logging.")
	}
}

func TestInvokeAPIOperationSet_FailBody(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	logger := log.New(buf, "", 0)
	_, err := InvokeAPIOperationSet(nil, nil, []byte("[{\"api\": \"destroy-all-humans\"}]"), logger)
	if err == nil {
		t.Error("Expected error.")
	} else if !strings.Contains(err.Error(), "Missing body") {
		t.Errorf("Wrong error: %s", err)
	}
	if buf.String() != "" {
		t.Error("Expected no logging.")
	}
}

func TestInvokeAPIOperationSet_Empty(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	logger := log.New(buf, "", 0)
	result, err := InvokeAPIOperationSet(nil, nil, []byte("[]"), logger)
	if err != nil {
		t.Error(err)
	} else if string(result) != "[]" {
		t.Errorf("Wrong result: %s", string(result))
	}
	if buf.String() != "" {
		t.Error("Expected no logging.")
	}
}


func TestInvokeAPIOperationSet_FailOperation(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	logger := log.New(buf, "", 0)
	_, err := InvokeAPIOperationSet(nil, &config.Context{}, []byte("[{\"api\": \"invalid-request\", \"body\": \"unused\"}]"), logger)
	if err == nil {
		t.Error("Expected error.")
	} else if !strings.Contains(err.Error(), "Could not find API request") {
		t.Errorf("Wrong error: %s", err)
	}
	if buf.String() != "" {
		t.Error("Expected no logging.")
	}
}
