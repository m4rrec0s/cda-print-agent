# CDA Print Agent

Agente desktop (Go + Wails + React) que monitora impressoras no Windows via WebSocket.

## ⚡ Início Rápido

```bash
# Instalar dependências
npm install --prefix frontend
go mod tidy

# Executar em desenvolvimento
wails dev
```

A app conecta automaticamente em `ws://localhost:3333/ws/print-agent` (configurável via `.env`).

## 📋 Funcionalidades

- ✅ Conexão WebSocket com reconexão automática (5s)
- ✅ Detecção de impressoras Windows (PowerShell)
- ✅ Interface React com status e log de eventos
- ✅ Janela frameless, always on top (400x500px)
- ✅ Eventos Wails em tempo real

## 🔌 Protocolo

**Servidor → Agente:**
```json
{"type": "CHECK_PRINTER"}
```

**Agente → Servidor:**
```json
{"type": "PRINTER_STATUS", "available": true, "printers": ["Impressora 1"], "timestamp": "15:30:45"}
```

## 📝 Configuração

Crie um arquivo `.env`:
```env
WS_URL=ws://localhost:3333/ws/print-agent
```

## 🔨 Build

```bash
wails build
# Executável em: build/bin/cda-print-agent.exe
```

## 📂 Arquivos

- `app.go` - Lógica principal, carrega .env, Wails bindings
- `websocket.go` - Gerenciador WebSocket com reconexão
- `printer.go` - Detecção de impressoras via PowerShell
- `main.go` - Inicialização, janela frameless/always-on-top
- `frontend/src/App.tsx` - Interface React
