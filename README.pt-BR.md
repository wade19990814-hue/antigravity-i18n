# antigravity-i18n

[English](./README.md) | [简体中文](./README.zh-CN.md) | [日本語](./README.ja.md) | [한국어](./README.ko.md) | [Español](./README.es.md) | [Deutsch](./README.de.md) | [Français](./README.fr.md) | Português do Brasil | [Русский](./README.ru.md)

Instale com um único comando um pacote de idioma para a interface do aplicativo de desktop [Google Antigravity](https://antigravity.google/) e restaure a versão oficial byte por byte quando quiser.

![Platform](https://img.shields.io/badge/platform-Windows%20%7C%20Linux%20%7C%20macOS-blue?style=flat-square)
![License](https://img.shields.io/badge/license-MIT-green?style=flat-square)
![Node](https://img.shields.io/badge/node-%3E%3D16-brightgreen?style=flat-square)

---

## Recursos

- **Sem instalação permanente**: execute diretamente com um único comando `npx`. O caminho de instalação é detectado e o aplicativo é reiniciado automaticamente.
- **Multilíngue**: todos os dados de idioma ficam em pacotes JSON em `src/locales/`. O mecanismo de tradução não contém dados específicos de nenhum idioma. Estão incluídos chinês simplificado, chinês tradicional, japonês, coreano, espanhol, alemão, francês, português do Brasil e russo.
- **Não invasivo**: somente a interface geral e os menus nativos são traduzidos. O editor de código (Monaco), o terminal (xterm) e as áreas de conversa não são alterados.
- **Restauração exata**: o `app.asar` original é salvo na primeira execução. `restore` recupera o arquivo oficial byte por byte.
- **Privado e offline**: não faz solicitações de rede, não inclui telemetria e não acessa tokens, sessões nem credenciais.
- **Plurais e direção de escrita**: textos com quantidades selecionam as formas plurais CLDR por meio de `Intl.PluralRules`; idiomas escritos da direita para a esquerda podem declarar sua direção.

---

## Uso

Requer Node.js 16 ou mais recente.

### Início rápido

```bash
# Aplicar português do Brasil
npx antigravity-i18n apply --locale pt-BR

# 简体中文: npx antigravity-i18n apply --locale zh-CN
# 繁體中文: npx antigravity-i18n apply --locale zh-Hant
# 日本語: npx antigravity-i18n apply --locale ja
# 한국어: npx antigravity-i18n apply --locale ko
# Español: npx antigravity-i18n apply --locale es
# Deutsch: npx antigravity-i18n apply --locale de
# Français: npx antigravity-i18n apply --locale fr
# Русский: npx antigravity-i18n apply --locale ru

# Restaurar a versão oficial
npx antigravity-i18n restore

# Verificar o idioma ativo e os backups
npx antigravity-i18n status

# Listar os pacotes de idioma incluídos
npx antigravity-i18n locales
```

`zh` é um atalho para `apply --locale zh-CN`, e `en` é um atalho para `restore`.

### Executar a partir do código-fonte

```bash
git clone https://github.com/wade19990814-hue/antigravity-i18n.git
cd antigravity-i18n
npm install
node bin/cli.js apply --locale pt-BR
```

### Opções

```text
Commands:
  apply             Instala um pacote de idioma (selecione com --locale)
  restore           Restaura a versão oficial sem tradução
  status            Mostra o idioma ativo e o caminho do aplicativo
  locales           Lista os pacotes de idioma incluídos

Options:
  --app-dir <path>  Caminho de instalação do Antigravity
  --locale <code>   Pacote de idioma a instalar (padrão: zh-CN)
  --no-restart      Não reinicia o Antigravity depois de aplicar o patch
  --no-kill         Exige que o Antigravity já esteja fechado e nunca encerra o processo
  --force           Encerra o aplicativo imediatamente, sem aguardar o fechamento normal
  -h, --help        Mostra a ajuda
  -v, --version     Mostra a versão
```

---

## Observações

1. **Salve seu trabalho**: a ferramenta aguarda até 20 segundos para que o aplicativo salve seu estado e seja fechado normalmente. Salve todo trabalho pendente antes de executar.
2. **Atualizações oficiais**: uma atualização do Antigravity substitui o `app.asar`. Execute `apply` novamente depois da atualização.
3. **Arquivos de backup**: a primeira execução cria `app.asar.clean-backup` no diretório `resources`. Depois de uma atualização oficial, ele é renovado automaticamente a partir do arquivo atual sem modificações. Não o exclua manualmente.
4. **Troca de idioma**: aplicar outro pacote substitui diretamente o pacote ativo; não é necessário executar `restore` antes.

---

## Contribuir

Correções de tradução e novos pacotes de idioma são bem-vindos. As chaves do dicionário são os textos originais em inglês da interface, portanto o pacote de referência também serve como inventário para um novo idioma.

```bash
# Exibir todos os textos em inglês que precisam ser traduzidos
node scripts/locale-report.js --keys

# Exibir cobertura, itens ausentes e valores ainda não traduzidos
node scripts/locale-report.js
```

Consulte [CONTRIBUTING.md](./CONTRIBUTING.md) para ver o formato dos pacotes e execute `npm test` antes de enviar uma Pull Request.

---

## Aviso legal

1. Use este projeto somente quando permitido pelas leis, pelos contratos e pelos termos aplicáveis do Antigravity. Você é responsável por verificar a conformidade do seu uso.
2. Esta é uma ferramenta independente de código aberto e não é afiliada, endossada nem autorizada pelo Google. Antigravity e as marcas relacionadas pertencem aos respectivos titulares.
3. A modificação do cliente é feita por sua conta e risco. Os autores não se responsabilizam por problemas inesperados, perda de dados ou outras consequências.

---

## Licença

[MIT](./LICENSE)
