package auction_entity

import (
	"context"
	"fullcycle-auction_go/internal/internal_error"
	"github.com/google/uuid"
	"time"
)

func CreateAuction(
	productName, category, description string,
	condition ProductCondition) (*Auction, *internal_error.InternalError) {
	auction := &Auction{
		Id:          uuid.New().String(),
		ProductName: productName,
		Category:    category,
		Description: description,
		Condition:   condition,
		Status:      Active,
		Timestamp:   time.Now(),
	}

	if err := auction.Validate(); err != nil {
		return nil, err
	}

	return auction, nil
}

func (au *Auction) Validate() *internal_error.InternalError {
	// Verifica se o nome do produto tem pelo menos 2 caracteres
	if len(au.ProductName) <= 1 {
		return internal_error.NewBadRequestError("product name too short")
	}
	
	// Verifica se a categoria tem pelo menos 3 caracteres
	if len(au.Category) <= 2 {
		return internal_error.NewBadRequestError("category too short")
	}
	
	// Verifica se a descrição tem pelo menos 11 caracteres
	if len(au.Description) <= 10 {
		return internal_error.NewBadRequestError("description too short")
	}
	
	// Verifica se a condição é válida
	if au.Condition != New && au.Condition != Refurbished && au.Condition != Used {
		return internal_error.NewBadRequestError("invalid product condition")
	}

	return nil
}

type Auction struct {
	Id          string
	ProductName string
	Category    string
	Description string
	Condition   ProductCondition
	Status      AuctionStatus
	Timestamp   time.Time
}

type ProductCondition int
type AuctionStatus int

const (
	Active AuctionStatus = iota
	Completed
)

const (
	New ProductCondition = iota + 1
	Used
	Refurbished
)

type AuctionRepositoryInterface interface {
	CreateAuction(
		ctx context.Context,
		auctionEntity *Auction) *internal_error.InternalError

	FindAuctions(
		ctx context.Context,
		status AuctionStatus,
		category, productName string) ([]Auction, *internal_error.InternalError)

	FindAuctionById(
		ctx context.Context, id string) (*Auction, *internal_error.InternalError)
}
