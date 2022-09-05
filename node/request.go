package node

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type txAddRequest struct {
	From  string `json:"from"`
	To    string `json:"to"`
	Value uint   `json:"value"`
	Data  string `json:"data"`
}

func requestFromBody(r *http.Request, target interface{}) error {
	reqBodyJson, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("unable to read request body. %s", err.Error())
	}

	defer r.Body.Close()

	if err = json.Unmarshal(reqBodyJson, target); err != nil {
		return fmt.Errorf("unable to unmarshal request body. %s", err.Error())
	}

	return nil
}
