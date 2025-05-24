package auction

import (
	"context"
	"fullcycle-auction_go/internal/entity/auction_entity"
	"fullcycle-auction_go/internal/internal_error"
	"os"
	"sync"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// TestAuctionAutoCloseInMemory realiza um teste do fechamento automático
// sem depender de um banco de dados externo
func TestAuctionAutoCloseInMemory(t *testing.T) {
	// Mock do repositório para teste
	mockRepo := setupInMemoryRepository()

	// Configura tempo curto para o teste
	os.Setenv("AUCTION_INTERVAL", "1s")

	// Cria um leilão para teste
	auction, err := auction_entity.CreateAuction(
		"Test Product",
		"Test Category",
		"Test Description for the product that needs to be at least 10 chars",
		auction_entity.New,
	)

	if err != nil {
		t.Fatalf("Failed to create auction entity: %v", err)
	}

	// Verifica estado inicial do leilão
	if auction.Status != auction_entity.Active {
		t.Errorf("Expected initial auction status to be Active, got %v", auction.Status)
	}

	// Registra o leilão no mapa de leilões ativos
	endTime := time.Now().Add(1 * time.Second)
	mockRepo.activeAuctionsMutex.Lock()
	mockRepo.activeAuctions[auction.Id] = endTime
	mockRepo.activeAuctionsMutex.Unlock()

	// Aguarda que o leilão expire
	time.Sleep(1500 * time.Millisecond)

	// Força a verificação de leilões expirados
	mockRepo.checkExpiredAuctions()

	// Verifica se o leilão foi removido do mapa (indicando que foi processado)
	mockRepo.activeAuctionsMutex.RLock()
	_, exists := mockRepo.activeAuctions[auction.Id]
	mockRepo.activeAuctionsMutex.RUnlock()

	if exists {
		t.Errorf("Expected auction to be removed from active auctions map")
	} else {
		t.Logf("Auction was successfully processed and removed from tracking")
	}
}

// Configura um repositório em memória para testes
func setupInMemoryRepository() *AuctionRepository {
	ctx, cancel := context.WithCancel(context.Background())

	// Cria um repositório com mock para testes
	mockRepo := &AuctionRepository{
		Collection:          nil, // Não precisa de uma coleção real para este teste
		activeAuctions:      make(map[string]time.Time),
		activeAuctionsMutex: &sync.RWMutex{},
		ctx:                 ctx,
		cancelFunc:          cancel,
	}

	// Substituímos a função updateAuctionStatus para evitar chamadas ao MongoDB
	mockRepo.updateAuctionStatus = func(id string, status auction_entity.AuctionStatus) *internal_error.InternalError {
		// Simulamos a atualização sem acessar o banco de dados
		return nil
	}

	return mockRepo
}

// TestAuctionAutoClose é um teste que requer um MongoDB local
// Este teste será pulado por padrão
func TestAuctionAutoClose(t *testing.T) {
	// Pula este teste porque depende de MongoDB externo
	// Remova esta linha para executar o teste quando tiver o MongoDB configurado
	t.Skip("Skipping test that requires MongoDB; run manually when MongoDB is available")

	// Configura variável de ambiente para um tempo curto para o teste
	os.Setenv("AUCTION_INTERVAL", "2s")

	// Conecta ao MongoDB - ajuste as credenciais conforme necessário
	ctx := context.Background()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		t.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer client.Disconnect(ctx)

	// Cria banco de teste
	database := client.Database("auction_test")

	// Limpa a coleção antes do teste
	_ = database.Collection("auctions").Drop(ctx)

	// Inicializa o repositório
	repo := NewAuctionRepository(database)

	// Cria um leilão para teste
	auction, err := auction_entity.CreateAuction(
		"Test Product",
		"Test Category",
		"Test Description for the product that is long enough",
		auction_entity.New,
	)
	if err != nil {
		t.Fatalf("Failed to create auction entity: %v", err)
	}

	// Insere o leilão
	err = repo.CreateAuction(ctx, auction)
	if err != nil {
		t.Fatalf("Failed to create auction: %v", err)
	}

	// Verifica se o leilão foi criado com status Active
	auctionFromDB, err := repo.FindAuctionById(ctx, auction.Id)
	if err != nil {
		t.Fatalf("Failed to find auction after creation: %v", err)
	}

	if auctionFromDB.Status != auction_entity.Active {
		t.Errorf("Expected auction status to be Active, got %v", auctionFromDB.Status)
	}

	// Aguarda mais que o tempo do leilão para ele expirar
	time.Sleep(3 * time.Second)

	// Verifica se o leilão foi fechado automaticamente
	auctionFromDB, err = repo.FindAuctionById(ctx, auction.Id)
	if err != nil {
		t.Fatalf("Failed to find auction after wait: %v", err)
	}

	if auctionFromDB.Status != auction_entity.Completed {
		t.Errorf("Expected auction status to be Completed after expiration, got %v", auctionFromDB.Status)
	}

	t.Logf("Auction was successfully closed automatically after expiration")
}
