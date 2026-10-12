# pull型カタログWebサービスのPoC

## 確認対象

`lumi-exporter`をHTTPサーバーとして起動し、登録されたlumiサービスをpullして、共通データと生レスポンスを同時に表示する。

| 登録先 | 公開API | 変換 |
|---|---|---|
| `lumitlink.aooba.net` | `/api/v1/entities/amamiyasaku` | `entity.v1`をそのまま利用 |
| `lumitree.aooba.net` | `/api/v1/calendars/ilife_official/events` | `event-candidate.v1`へ変換 |
| `lumidive.aooba.net` | `/api/v1/events/plkt1022` | `event-candidate.v1`へ変換 |

`lumitree`はカレンダーイベント一覧、`lumidive`は単一イベント詳細という違いがあるため、登録`kind`ごとに薄いadapterを持たせた。サービスrepoを直接importせず、HTTP境界のJSONだけを読むことで結合を抑えている。

## 完了条件

- 登録ソースの一覧をAPIで確認できる
- 全登録ソースを一回のpullで取得できる
- `entity.v1`または`event-candidate.v1`として正規化された値が画面に表示される
- HTTPステータス、取得時間、endpoint、上流の生JSONを画面で確認できる
- `kubectl logs -f`などの外部ログ監視でリクエストを追跡できる

## 未実装と次段階

- `lumibot`のadapter
- 定期poll、永続化、差分更新
- endpoint許可リスト、認証、SSRF防御
- `lumitree`と`lumidive`が直接`lumi-spec`のfeed envelopeを返すための各repo側変更

参考:

- [lumitree OpenAPI](https://github.com/AobaIwaki123/lumitree/blob/main/api/openapi.yaml)
- [lumidive OpenAPI](https://github.com/AobaIwaki123/lumidive/blob/main/api/openapi.yaml)
- [JSON Schema 2020-12](https://json-schema.org/draft/2020-12/json-schema-core)
- [OWASP SSRF Prevention Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Server_Side_Request_Forgery_Prevention_Cheat_Sheet.html)
