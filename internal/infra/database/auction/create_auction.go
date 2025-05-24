package auction

import (
	"context"
	"fmt"
	"fullcycle-auction_go/configuration/logger"
	"fullcycle-auction_go/internal/entity/auction_entity"
	"fullcycle-auction_go/internal/internal_error"
	"os"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type AuctionEntityMongo struct {
	Id          string                          `bson:"_id"`
	ProductName string                          `bson:"product_name"`
	Category    string                          `bson:"category"`
	Description string                          `bson:"description"`
	Condition   auction_entity.ProductCondition `bson:"condition"`
	Status      auction_entity.AuctionStatus    `bson:"status"`
	Timestamp   int64                           `bson:"timestamp"`
}

type AuctionRepository struct {
	Collection *mongo.Collection
	// Para manter o controle de leilões em andamento
	activeAuctions      map[string]time.Time
	activeAuctionsMutex *sync.RWMutex
	// Contexto para gerenciar o ciclo de vida das goroutines
	ctx        context.Context
	cancelFunc context.CancelFunc
	// Função para atualizar status do leilão - pode ser substituída em testes
	updateAuctionStatus func(id string, status auction_entity.AuctionStatus) *internal_error.InternalError
}

func NewAuctionRepository(database *mongo.Database) *AuctionRepository {
	ctx, cancel := context.WithCancel(context.Background())
	repo := &AuctionRepository{
		Collection:         database.Collection("auctions"),
		activeAuctions:     make(map[string]time.Time),
		activeAuctionsMutex: &sync.RWMutex{},
		ctx:                ctx,
		cancelFunc:         cancel,
	}
	
	// Define a função padrão para atualizar o status
	repo.updateAuctionStatus = repo.updateAuctionStatusImpl
	
	// Inicia a goroutine para monitorar e fechar leilões expirados
	go repo.monitorAuctions()
	
	return repo
}

// Função que monitora os leilões ativos e fecha aqueles que expiraram
func (ar *AuctionRepository) monitorAuctions() {
	logger.Info("Starting auction monitoring routine")
	
	// Intervalo de verificação (por padrão a cada 5 segundos)
	interval := getCheckInterval()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ar.ctx.Done():
			logger.Info("Stopping auction monitoring routine")
			return
		case <-ticker.C:
			ar.checkExpiredAuctions()
		}
	}
}

// Verifica e fecha leilões expirados
func (ar *AuctionRepository) checkExpiredAuctions() {
	now := time.Now()
	var expiredAuctionIds []string
	
	// Coleta os IDs de leilões expirados com lock de leitura
	ar.activeAuctionsMutex.RLock()
	for id, endTime := range ar.activeAuctions {
		if now.After(endTime) {
			expiredAuctionIds = append(expiredAuctionIds, id)
		}
	}
	ar.activeAuctionsMutex.RUnlock()
	
	// Processa cada leilão expirado
	for _, id := range expiredAuctionIds {
		// Remove do mapa com lock de escrita
		ar.activeAuctionsMutex.Lock()
		delete(ar.activeAuctions, id)
		ar.activeAuctionsMutex.Unlock()
		
		// Atualiza o status no banco de dados
		err := ar.updateAuctionStatus(id, auction_entity.Completed)
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to close expired auction: %s", id), err)
		} else {
			logger.Info(fmt.Sprintf("Successfully closed expired auction: %s", id))
		}
	}
}

// Implementação real da atualização de status no banco de dados
func (ar *AuctionRepository) updateAuctionStatusImpl(id string, status auction_entity.AuctionStatus) *internal_error.InternalError {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	filter := bson.M{"_id": id}
	update := bson.M{"$set": bson.M{"status": status}}
	
	_, err := ar.Collection.UpdateOne(ctx, filter, update)
	if err != nil {
		logger.Error(fmt.Sprintf("Error updating auction status for id=%s", id), err)
		return internal_error.NewInternalServerError("Error updating auction status")
	}
	
	return nil
}

// Calcula o intervalo de duração do leilão com base na variável de ambiente
func getAuctionDuration() time.Duration {
	auctionInterval := os.Getenv("AUCTION_INTERVAL")
	duration, err := time.ParseDuration(auctionInterval)
	if err != nil {
		return time.Minute * 5 // Valor padrão: 5 minutos
	}
	
	return duration
}

// Calcula o intervalo de verificação para fechar leilões
func getCheckInterval() time.Duration {
	// Por padrão, verifica a cada 5 segundos
	return time.Second * 5
}

func (ar *AuctionRepository) CreateAuction(
	ctx context.Context,
	auctionEntity *auction_entity.Auction) *internal_error.InternalError {
	auctionEntityMongo := &AuctionEntityMongo{
		Id:          auctionEntity.Id,
		ProductName: auctionEntity.ProductName,
		Category:    auctionEntity.Category,
		Description: auctionEntity.Description,
		Condition:   auctionEntity.Condition,
		Status:      auctionEntity.Status,
		Timestamp:   auctionEntity.Timestamp.Unix(),
	}
	_, err := ar.Collection.InsertOne(ctx, auctionEntityMongo)
	if err != nil {
		logger.Error("Error trying to insert auction", err)
		return internal_error.NewInternalServerError("Error trying to insert auction")
	}
	
	// Adiciona o leilão ao mapa de leilões ativos com seu tempo de expiração
	endTime := auctionEntity.Timestamp.Add(getAuctionDuration())
	
	ar.activeAuctionsMutex.Lock()
	ar.activeAuctions[auctionEntity.Id] = endTime
	ar.activeAuctionsMutex.Unlock()
	
	logger.Info(fmt.Sprintf("Auction created with ID: %s, will expire at: %s", 
		auctionEntity.Id, endTime.Format(time.RFC3339)))
	
	return nil
}
