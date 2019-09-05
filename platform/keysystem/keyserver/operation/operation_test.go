package operation

import (
	"bytes"
	"log"
	"strings"
	"testing"

	"github.com/sipb/homeworld/platform/keysystem/keyserver/account"
	"github.com/sipb/homeworld/platform/keysystem/keyserver/config"
)

func TestInvokeAPIOperation_NoAPI(t *testing.T) {
	opctx := account.OperationContext{Account: &account.Account{Principal: "test-account"}}
	gctx := config.Context{}
	buf := bytes.NewBuffer(nil)
	logger := log.New(buf, "", 0)
	_, err := InvokeAPIOperation(&opctx, &gctx, "test-api", "gemstone", logger)
	if err == nil {
		t.Error("Expected error.")
	} else if !strings.Contains(err.Error(), "could not find API request") {
		t.Error("Wrong error.")
	}
	if buf.String() != "" {
		t.Error("Expected no logging.")
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
	} else if !strings.Contains(err.Error(), "missing API") {
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
	} else if !strings.Contains(err.Error(), "missing body") {
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
	} else if !strings.Contains(err.Error(), "could not find API request") {
		t.Errorf("Wrong error: %s", err)
	}
	if buf.String() != "" {
		t.Error("Expected no logging.")
	}
}
