Come usarlo passo passo 

Installa DCMTK (deve esserci storescp nel PATH) 

Linux (Debian/Ubuntu): sudo apt install dcmtk 

macOS (brew): brew install dcmtk 

Windows: installer DCMTK e aggiungi alla PATH. 

Metti il file main.go in una cartella. 

Crea la cartella di storage (se vuoi una diversa dalla default): 

mkdir dicom-inbox Eseguilo (su Linux/macOS con porta 104 → serve sudo): 
sudo go run main.go \ -aet MYPACS \ -port 104 \ -storage ./dicom-inbox \ -storescp storescp 

Configura l’ecografo con: 
AE Title remoto: MYPACS 
IP remoto: IP del tuo PC/server Porta: 104 
Quando l’eco invia immagini: storescp (pilotato da Go) riceve i C-STORE i file DICOM finiscono nella cartella ./dicom-inbox
