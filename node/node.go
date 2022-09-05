package node

import (
	"fmt"
	"net/http"
	"the-blockchain-bar/database"
)

const httpPort = 8080

func Run(dataDir string) error {
	fmt.Println(fmt.Sprintf("Listening on HTTP port: %d", httpPort))

	state, err := database.NewStateFromDisk(dataDir)
	if err != nil {
		return err
	}

	defer state.Close()

	http.HandleFunc("/balances/list", func(w http.ResponseWriter, r *http.Request) {
		listBalancesHandler(w, r, state)
	})

	http.HandleFunc("/tx/add", func(w http.ResponseWriter, r *http.Request) {
		txAddHandler(w, r, state)
	})

	http.HandleFunc("/node/status", func(w http.ResponseWriter, r *http.Request) {
		statusHandler(w, r, state)
	})

	return http.ListenAndServe(fmt.Sprintf(":%d", httpPort), nil)
}

func listBalancesHandler(w http.ResponseWriter, _ *http.Request, state *database.State) {
	writeSuccessfulResponse(w, balancesResponse{
		Hash:     state.LatestBlockHash(),
		Balances: state.Balances,
	})
}

func txAddHandler(w http.ResponseWriter, r *http.Request, state *database.State) {
	req := txAddRequest{}
	if err := requestFromBody(r, &req); err != nil {
		writeErrorResponse(w, err)

		return
	}

	tx := database.NewTx(database.NewAccount(req.From), database.NewAccount(req.To), req.Value, req.Data)

	if err := state.AddTx(tx); err != nil {
		writeErrorResponse(w, err)

		return
	}

	hash, err := state.Persist()
	if err != nil {
		writeErrorResponse(w, err)

		return
	}

	writeSuccessfulResponse(w, txAddResponse{hash})
}

func statusHandler(w http.ResponseWriter, r *http.Request, state *database.State) {
	res := statusResponse{
		Hash:   state.LatestBlockHash(),
		Number: state.LatestBlock().Header.Number,
	}

	writeSuccessfulResponse(w, res)
}
