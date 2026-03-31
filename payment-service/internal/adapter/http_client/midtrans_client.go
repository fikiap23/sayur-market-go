package httpclient

import (
	"payment-service/config"

	"github.com/labstack/gommon/log"
	"github.com/midtrans/midtrans-go"
	"github.com/midtrans/midtrans-go/snap"
)

type MidtransClientInterface interface {
	CreateTransaction(orderID string, amount int64, customerName, customerEmail string) (string, error)
}

type midtransClient struct {
	cfg *config.Config
}

// CreateTransaction implements MidtransClientInterface.
func (m *midtransClient) CreateTransaction(orderID string, amount int64, customerName string, customerEmail string) (string, error) {
	midtrans.ServerKey = m.cfg.Midtrans.ServerKey
	midtrans.Environment = midtrans.EnvironmentType(m.cfg.Midtrans.Environment)

	snapReq := &snap.Request{
		TransactionDetails: midtrans.TransactionDetails{
			OrderID:  orderID,
			GrossAmt: amount,
		},
		CustomerDetail: &midtrans.CustomerDetails{
			FName: customerName,
			Email: customerEmail,
		},
	}

	snapRes, err := snap.CreateTransaction(snapReq)
	if err != nil {
		log.Errorf("[MidtransClient-1] Failed to create transaction: %v", err)
		return "", err
	}

	return snapRes.Token, nil
}

func NewMidtransClient(cfg *config.Config) MidtransClientInterface {
	return &midtransClient{cfg: cfg}
}
