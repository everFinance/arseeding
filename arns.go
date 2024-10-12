package arseeding

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/everFinance/arseeding/schema"
	"github.com/everFinance/goether"
	"github.com/permadao/goao"
	goarSchema "github.com/permadao/goar/schema"
	"strings"
)

const (
	ArNsPid = "agYcCFJtrMG6cqMuZfskIkFTGvUPddICmtQSBIoPdiA"
)

type ArNs struct {
	aoCli *goao.Client
}

func NewArNs(cuUrl, muUrl string) *ArNs {
	eccSigner, _ := goether.NewSigner("4c3f9a1e5b234ce8f1ab58d82f849c0f70a4d5ceaf2b6e2d9a6c58b1f897ef0a")
	aoCli, err := goao.NewClient(muUrl, cuUrl, eccSigner)
	if err != nil {
		panic(err)
	}
	return &ArNs{aoCli: aoCli}
}

func (a *ArNs) GetRootDomainRecord(rootDomain string) (record *schema.DomainRecord, err error) {
	resp, err := a.aoCli.DryRun(ArNsPid, "", []goarSchema.Tag{
		{Name: "Action", Value: "Record"},
		{Name: "Name", Value: rootDomain},
	})
	if err != nil {
		return nil, err
	}
	if len(resp.Messages) != 1 {
		return nil, errors.New("empty messages")
	}

	res := resp.Messages[0].(map[string]interface{})
	err = json.Unmarshal([]byte(fmt.Sprintf("%v", res["Data"])), &record)
	return
}

func (a *ArNs) GetDomainProcessState(pid string) (state *schema.DomainState, err error) {
	resp, err := a.aoCli.DryRun(pid, "", []goarSchema.Tag{
		{Name: "Action", Value: "State"},
	})
	if err != nil {
		return
	}
	if len(resp.Messages) != 1 {
		return nil, errors.New("empty messages")
	}
	res := resp.Messages[0].(map[string]interface{})
	err = json.Unmarshal([]byte(fmt.Sprintf("%v", res["Data"])), &state)
	return
}

func (a *ArNs) QueryDomainTxId(domain string) (string, error) {
	spliteDomains := strings.Split(domain, "_")
	if len(spliteDomains) > 2 { // todo now only support level-2 subdomain
		return "", errors.New("current arseeding gw not support over level-2 subdomain")
	}
	rootDomain := spliteDomains[len(spliteDomains)-1]
	// step1 query root domain process state
	record, err := a.GetRootDomainRecord(rootDomain)
	if err != nil {
		return "", err
	}
	state, err := a.GetDomainProcessState(record.ProcessId)
	if err != nil {
		return "", err
	}

	// step2 query latest txId
	// Currently, only level-1 domain name resolution is queried
	subdomain := spliteDomains[0]
	if subdomain == rootDomain {
		subdomain = "@"
	}
	res, ok := state.Records[subdomain]
	if !ok {
		return "", errors.New("not exist record")
	}
	return res.TransactionId, nil
}
