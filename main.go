package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"
)

func __main() {
	fmt.Println("Hello from main package")
}

// Config del nostro "server"
type ServerConfig struct {
	AETitle    string
	Port       string
	StorageDir string
	StoreScp   string // path a storescp (DCMTK)
}

func main() {
	// --- Flags ---
	aeTitle := flag.String("aet", "MYPACS", "AE Title del server DICOM (SCP)")
	port := flag.String("port", "3000", "Porta DICOM in ascolto (es. 3000)")
	storageDir := flag.String("storage", "./dicom-inbox", "Directory dove salvare i file DICOM ricevuti")
	storeScpPath := flag.String("storescp", "storescp", "Path al binario storescp (DCMTK)")

	flag.Parse()

	cfg := ServerConfig{
		AETitle:    *aeTitle,
		Port:       *port,
		StorageDir: *storageDir,
		StoreScp:   *storeScpPath,
	}

	// --- Setup logging ---
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	log.Println("Avvio server DIMSE (wrapper DCMTK storescp)")
	log.Printf("AE Title: %s", cfg.AETitle)
	log.Printf("Porta   : %s", cfg.Port)
	log.Printf("Storage : %s", cfg.StorageDir)
	log.Printf("storescp: %s", cfg.StoreScp)

	// Crea la cartella di storage se non esiste
	if err := os.MkdirAll(cfg.StorageDir, 0755); err != nil {
		log.Fatalf("Impossibile creare la cartella di storage: %v", err)
	}

	// Controllo base sulla porta 3000 su sistemi *nix
	if cfg.Port == "3000" && (runtime.GOOS == "linux" || runtime.GOOS == "darwin") {
		log.Println("ATTENZIONE: la porta 3000 richiede permessi elevati su Linux/macOS (usa sudo o cambia porta).")
	}

	// Context per gestire stop/graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Avvia storescp
	cmd, err := startStoreScp(ctx, cfg)
	if err != nil {
		log.Fatalf("Errore nell'avviare storescp: %v", err)
	}

	// Gestione segnali (Ctrl+C, SIGTERM)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		log.Printf("Ricevuto segnale %s: stop del server DICOM...", sig)
		// Cancella il contesto (chiude storescp)
		cancel()

		// Prova a terminare il processo gentile
		if cmd.Process != nil {
			_ = cmd.Process.Signal(syscall.SIGTERM)
			// Se dopo un po' è ancora vivo, kill
			go func() {
				time.Sleep(3 * time.Second)
				_ = cmd.Process.Kill()
			}()
		}
	}()

	// Attende la fine di storescp
	err = cmd.Wait()
	if err != nil {
		// Se il contesto è stato cancellato, è uno shutdown voluto
		select {
		case <-ctx.Done():
			log.Println("storescp terminato (shutdown richiesto).")
		default:
			log.Printf("storescp terminato con errore: %v", err)
		}
	} else {
		log.Println("storescp terminato correttamente.")
	}

	log.Println("Server DIMSE Go/DCMTK chiuso.")
}

// startStoreScp avvia il processo storescp (DCMTK) con i parametri richiesti
func startStoreScp(ctx context.Context, cfg ServerConfig) (*exec.Cmd, error) {
	absStorageDir, err := filepath.Abs(cfg.StorageDir)
	if err != nil {
		return nil, fmt.Errorf("errore nel risolvere il path di storage: %w", err)
	}

	// Argomenti per storescp:
	//  -d           → aumentare logging (DEBUG)
	//  -aet AE      → AE Title del server
	//  -od dir      → output directory (dove salvare i file ricevuti)
	//  -xf logfile  → opzionale: log su file (puoi aggiungerlo)
	//  port         → porta in ascolto
	args := []string{
		"-d",                // log verbose su stdout
		"-aet", cfg.AETitle, // AE Title del server
		"-od", absStorageDir,
		cfg.Port,
	}

	cmd := exec.CommandContext(ctx, cfg.StoreScp, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Printf("Eseguo: %s %s", cfg.StoreScp, stringsJoin(args, " "))

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("errore nell'avvio di storescp: %w", err)
	}

	log.Printf("storescp avviato (PID=%d), in ascolto sulla porta %s, AE=%s, storage=%s",
		cmd.Process.Pid, cfg.Port, cfg.AETitle, absStorageDir)

	return cmd, nil
}

// stringsJoin è una piccola utility per evitare di importare strings solo per Join
func stringsJoin(ss []string, sep string) string {
	switch len(ss) {
	case 0:
		return ""
	case 1:
		return ss[0]
	}
	n := len(sep) * (len(ss) - 1)
	for i := 0; i < len(ss); i++ {
		n += len(ss[i])
	}
	b := make([]byte, n)
	bp := copy(b, ss[0])
	for _, s := range ss[1:] {
		bp += copy(b[bp:], sep)
		bp += copy(b[bp:], s)
	}
	return string(b)
}
