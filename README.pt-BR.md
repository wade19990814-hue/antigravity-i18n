# antigravity-i18n


Ferramenta de localização de IU para o cliente de desktop Google Antigravity.


[简体中文](./README.zh-CN.md) | [English](./README.md) | [繁體中文](./README.zh-Hant.md) | [日本語](./README.ja.md) | [한국어](./README.ko.md) | [Deutsch](./README.de.md) | [Español](./README.es.md) | [Français](./README.fr.md) | **Português (Brasil)** | [Русский](./README.ru.md)

## Status do projeto e plataformas compatíveis

Este repositório é o principal local de desenvolvimento daqui em diante. A branch `main` atual contém o aplicativo Go/Wails v3 e, por enquanto, oferece suporte **apenas ao Windows**. A implementação anterior multiplataforma em Node.js/npm é preservada separadamente na [branch legacy-node](https://github.com/wade19990814-hue/antigravity-i18n/tree/legacy-node); suas tags e versões v1/v2 permanecem inalteradas.

---

Um executável independente que reúne uma interface gráfica moderna, um mecanismo Go asar puro e pacotes de idiomas completos. Funciona imediatamente com zero dependências de Node.js ou bibliotecas de tempo de execução e oferece suporte à restauração exata de bytes para a compilação oficial a qualquer momento.

---

## Principais recursos

- **Binário independente único**: executável independente de aproximadamente 12 MB — não é necessário tempo de execução de Node.js, npm ou Python.
- **Modo duplo GUI e CLI**: abre uma janela limpa da área de trabalho por padrão, enquanto incorpora subcomandos CLI completos (`apply` / `restore` / `status` / `locales`) para scripts e automação.
- **Restauração exata de bytes**: Cria automaticamente um backup original (`app.asar.clean-backup`) antes de qualquer alteração. A restauração recupera um arquivo idêntico byte por byte (verificado pelo SHA-256) ao lançamento oficial.
- **Design não invasivo**: localiza apenas a UI do shell, menus nativos e caixas de diálogo de encerramento. Editores de código (Monaco), terminais integrados (xterm) e fluxos de conversação de IA permanecem estritamente intocados.
- **Off-line e confiável**: Zero atividade de rede, zero coleta de dados. Equipado com validação antecipada de sintaxe JavaScript e reversão transacional em caso de erros inesperados.

---

## Download e uso

### Download

Baixe o `antigravity-i18n.exe` mais recente na página de lançamentos e execute-o diretamente.

### 1. Interface Gráfica (Recomendado)

Clique duas vezes em `antigravity-i18n.exe`:
1. O aplicativo detecta automaticamente a pasta de instalação do Antigravity (ou clique em ‘Alterar local’ para especificar manualmente).
2. Clique no seu idioma preferido (selecionar inglês restaura a versão oficial).
3. Clique em ‘Aplicar’; a ferramenta confirma o encerramento do processo, atualiza os recursos e reinicia o Antigravity.

### 2. Interface de linha de comando

O mesmo binário pode ser invocado no terminal:

```powershell
# Check current localization status and backups
antigravity-i18n.exe status

# Install a language pack (Simplified Chinese by default)
antigravity-i18n.exe apply --locale zh-CN

# Restore the official untranslated English version
antigravity-i18n.exe restore

# List all bundled language packs with coverage
antigravity-i18n.exe locales
```

Opções comuns:
- `--app-dir <caminho>`: Especifique manualmente o diretório de instalação do Antigravity (ou defina via `ANTIGRAVITY_APP_DIR`).
- `--locale <code>`: Código do idioma de destino (por exemplo, `zh-CN`, `zh-Hant`, `ja`, `ko`, `de`, `es`, `fr`).
- `--no-restart`: Não reinicie o Antigravity após a correção.
- `--force`: Ignore a espera de desligamento normal de 30 segundos e encerre os processos imediatamente.
- `--yes`: Suprime o prompt de confirmação de desligamento.

---

## Idiomas incluídos

Todos os pacotes de idiomas são incorporados diretamente no executável:

| Código | Idioma | Cobertura | Notas |
| :--- | :--- | :--- | :--- |
| `zh-CN` | Simplified Chinese (简体中文) | 100% | Cobertura completa da interface, menus nativos e diálogos |
| `zh-Hant` | Traditional Chinese (繁體中文) | 100% | Cobertura completa com terminologia técnica verificada |
| `ja` | Japanese (日本語) | 100% | Cobertura completa com terminologia técnica verificada |
| `ko` | Korean (한국어) | 100% | Cobertura completa com terminologia técnica verificada |
| `de` | German (Deutsch) | 100% | Cobertura completa com terminologia técnica verificada |
| `es` | Spanish (Español) | 100% | Cobertura completa com terminologia técnica verificada |
| `fr` | French (Français) | 100% | Cobertura completa com terminologia técnica verificada |
| `pt-BR` | Portuguese (Brasil) | 100% | Cobertura completa com terminologia técnica verificada |
| `ru` | Russian (Русский) | 100% | Cobertura completa com terminologia técnica verificada |
| `en` | English | - | Build oficial intocado (selecione para restaurar) |

---

## Perguntas frequentes e mecânica

### 1. O que acontece após uma atualização oficial do Antigravity?
As atualizações oficiais substituirão `app.asar`. Basta reabrir esta ferramenta e clicar em ‘Aplicar’ novamente. A ferramenta atualizará automaticamente seu backup limpo usando a nova versão oficial.

### 2. Troca e restauração de idioma
Você pode alternar diretamente entre qualquer idioma; a ferramenta substitui perfeitamente os patches ativos sem exigir uma restauração manual primeiro. Clicar em 'Restaurar Inglês Oficial' retorna o cliente a um estado intocado a qualquer momento.

### 3. Gerenciamento de arquivos de backup
- `app.asar.clean-backup`: O backup oficial original necessário para restauração. Não exclua manualmente.
- `app.asar.bak-*`: Instantâneos históricos com carimbo de data/hora deixados ao lado do arquivo. Depois que sua instalação atual for verificada como estável, você poderá excluir esses arquivos com segurança para liberar espaço em disco.

---

## Construindo a partir da fonte

**Pré-requisitos**:
- Vá para 1,26+
- Lamentos v2 CLI

```powershell
# Clone the repository
git clone https://github.com/wade19990814-hue/antigravity-i18n.git
cd antigravity-i18n

# Build both GUI and CLI binaries
.\build.ps1

# Run tests (including end-to-end asar verification)
$env:ANTIGRAVITY_ASAR = "$env:LOCALAPPDATA\Programs\antigravity\resources\app.asar.clean-backup"
go test ./...
```

---

## Isenção de responsabilidade

1. Este é um projeto não oficial mantido pela comunidade e não é afiliado, endossado ou autorizado pelo Google ou pela Antigravity.
2. Google Antigravity e marcas registradas relacionadas pertencem aos seus respectivos proprietários.
3. Esta ferramenta é fornecida para uso pessoal e educacional. Garanta a conformidade com os termos de serviço do software relevantes.

## Licença

[Licença MIT](./LICENSE)
