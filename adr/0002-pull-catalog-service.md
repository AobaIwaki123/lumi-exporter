# ADR 0002: 登録済みlumiサービスをpullする薄いWebサービス

## ステータス

採用予定

## 背景

`lumi-spec`の契約を複数の実サービスで使えることを、イベントfixtureだけでなく実際のHTTP連携と画面表示で確認したい。`lumitree`、`lumidive`、`lumitlink`はそれぞれ固有のAPIレスポンスを持つため、exporterが登録先をpullし、共通契約へ投影した結果を一つの画面で比較できると検証しやすい。

## 決定

`lumi-exporter`に薄いWebモードを追加する。

- 登録情報はJSONファイルで管理し、`service`、`kind`、`endpoint`、`enabled`だけを持つ。
- `entity` / `entity_feed` / `event_feed`は`lumi-spec`のschema versionを検査して受け取る。
- `lumitree_calendar`と`lumidive_event`は各サービスの公開APIレスポンスを`event-candidate.v1`へ変換するadapterとする。
- `GET /api/v1/catalog`は全登録先を並行pullし、HTTPステータス、処理時間、エラー、生JSON、正規化データを返す。
- UIはこのレスポンスをそのまま描画し、モックデータを持たない。

## 捨てたものとリスク

初期版ではDB、認証、バックグラウンドpoll、キュー、サービス間の直接SDK依存を採用しない。設定ファイルを変更して再起動する運用になるが、PoCの責務と検証経路を小さく保てる。

endpointを自由に設定できるため、外部公開する場合はSSRF、認証情報、リクエスト先の許可リストを追加で設計する必要がある。このWebモードは初期段階では管理下のネットワークで使う。

`lumibot`は将来のpull対象候補として記録するが、初期登録とadapter実装には含めない。

参考:

- [lumitree repository](https://github.com/AobaIwaki123/lumitree)
- [lumidive repository](https://github.com/AobaIwaki123/lumidive)
- [lumi-spec repository](https://github.com/AobaIwaki123/lumi-spec)
- [OWASP SSRF Prevention Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Server_Side_Request_Forgery_Prevention_Cheat_Sheet.html)
