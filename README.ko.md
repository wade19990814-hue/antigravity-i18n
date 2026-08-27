# antigravity-i18n

[English](./README.md) | [简体中文](./README.zh-CN.md) | [日本語](./README.ja.md) | 한국어 | [Español](./README.es.md) | [Deutsch](./README.de.md) | [Français](./README.fr.md) | [Português do Brasil](./README.pt-BR.md) | [Русский](./README.ru.md)

한 줄 명령으로 [Google Antigravity](https://antigravity.google/) 데스크톱 앱에 UI 언어 팩을 적용하고, 언제든 공식 버전을 바이트 단위로 정확히 복원할 수 있습니다.

![Platform](https://img.shields.io/badge/platform-Windows%20%7C%20Linux%20%7C%20macOS-blue?style=flat-square)
![License](https://img.shields.io/badge/license-MIT-green?style=flat-square)
![Node](https://img.shields.io/badge/node-%3E%3D16-brightgreen?style=flat-square)

---

## 주요 기능

- **별도 설치 불필요**: 하나의 `npx` 명령으로 바로 실행하며, 설치 경로 감지와 앱 재시작도 자동으로 처리합니다.
- **다국어 지원**: 언어 데이터는 모두 `src/locales/` 아래의 JSON 팩으로 분리되어 있습니다. 번역 엔진 자체는 특정 언어에 의존하지 않습니다. 중국어 간체, 일본어, 한국어, 스페인어, 독일어, 프랑스어, 브라질 포르투갈어, 러시아어가 기본 제공됩니다.
- **비침습적**: 셸 UI와 네이티브 메뉴만 번역합니다. 코드 편집기(Monaco), 터미널(xterm), 대화 영역은 변경하지 않습니다.
- **바이트 단위 복원**: 처음 실행할 때 원본 `app.asar`를 백업하며, `restore`로 공식 아카이브를 변경 전 상태 그대로 되돌립니다.
- **오프라인 및 개인정보 보호**: 네트워크 요청과 원격 측정이 없으며 토큰, 세션, 자격 증명에 접근하지 않습니다.
- **복수형과 쓰기 방향 지원**: 수량이 포함된 문자열은 `Intl.PluralRules`로 CLDR 복수형을 선택하고, 오른쪽에서 왼쪽으로 쓰는 언어는 쓰기 방향을 선언할 수 있습니다.

---

## 사용법

Node.js 16 이상이 필요합니다.

### 빠른 시작

```bash
# 한국어 적용
npx antigravity-i18n apply --locale ko

# 简体中文: npx antigravity-i18n apply --locale zh-CN
# 日本語: npx antigravity-i18n apply --locale ja
# Español: npx antigravity-i18n apply --locale es
# Deutsch: npx antigravity-i18n apply --locale de
# Français: npx antigravity-i18n apply --locale fr
# Português do Brasil: npx antigravity-i18n apply --locale pt-BR
# Русский: npx antigravity-i18n apply --locale ru

# 공식 버전으로 복원
npx antigravity-i18n restore

# 현재 언어와 백업 상태 확인
npx antigravity-i18n status

# 기본 제공 언어 팩 목록 표시
npx antigravity-i18n locales
```

`zh`는 `apply --locale zh-CN`의 단축 명령이고, `en`은 `restore`의 단축 명령입니다.

### 소스에서 실행

```bash
git clone https://github.com/wade19990814-hue/antigravity-i18n.git
cd antigravity-i18n
npm install
node bin/cli.js apply --locale ko
```

### 옵션

```text
Commands:
  apply             언어 팩 적용(--locale로 선택)
  restore           번역되지 않은 공식 버전으로 복원
  status            현재 언어와 앱 경로 표시
  locales           기본 제공 언어 팩 목록 표시

Options:
  --app-dir <path>  Antigravity 설치 경로
  --locale <code>   적용할 언어 팩(기본값: zh-CN)
  --no-restart      패치 후 앱을 다시 시작하지 않음
  --no-kill         앱이 이미 종료되어 있어야 하며 프로세스를 종료하지 않음
  --force           정상 종료 대기 없이 즉시 앱을 종료
  -h, --help        도움말 표시
  -v, --version     버전 표시
```

---

## 주의 사항

1. **작업을 저장하세요**: 앱이 상태를 저장하고 정상 종료할 수 있도록 최대 20초 동안 기다립니다. 실행하기 전에 저장하지 않은 작업을 저장하세요.
2. **공식 업데이트**: Antigravity 업데이트는 `app.asar`를 덮어씁니다. 업데이트 후 `apply`를 다시 실행하세요.
3. **백업 파일**: 처음 실행할 때 `resources` 폴더에 `app.asar.clean-backup`을 만듭니다. 공식 업데이트 후에는 현재의 수정되지 않은 아카이브로 자동 갱신됩니다. 직접 삭제하지 마세요.
4. **언어 전환**: 다른 언어 팩을 적용하면 현재 팩을 바로 교체합니다. 먼저 `restore`를 실행할 필요가 없습니다.

---

## 기여

번역 수정과 새로운 언어 팩 추가를 환영합니다. 사전의 키는 번역되지 않은 영어 UI 문자열이므로, 기본 제공 팩은 새 언어를 작성할 때 원문 목록으로도 사용됩니다.

```bash
# 번역해야 할 모든 영어 문자열 출력
node scripts/locale-report.js --keys

# 적용 범위, 누락 항목, 미번역 자리표시자 확인
node scripts/locale-report.js
```

언어 팩 형식은 [CONTRIBUTING.md](./CONTRIBUTING.md)를 참고하고, Pull Request를 보내기 전에 `npm test`를 실행하세요.

---

## 면책 조항

1. 관련 법률, 계약 및 Antigravity 이용 약관에서 허용하는 범위에서만 이 프로젝트를 사용하세요. 사용자는 자신의 사용 방식이 적법하고 규정을 준수하는지 확인할 책임이 있습니다.
2. 이 프로젝트는 독립적인 오픈 소스 도구이며 Google과 제휴하거나 Google의 보증 또는 승인을 받은 것이 아닙니다. Antigravity 및 관련 상표는 각 권리자에게 귀속됩니다.
3. 클라이언트 수정에 따른 위험은 사용자가 부담합니다. 작성자는 예상치 못한 문제, 데이터 손실 또는 기타 결과에 대해 책임지지 않습니다.

---

## 라이선스

[MIT](./LICENSE)
