# lumi-exporter

`lumitree`、`lumidive`などから得たイベント候補を、`lumi-spec`の共通契約で比較する名寄せPoCです。

初期版は候補生成や外部API取得を含まないCLIです。ラベル付きfixtureを入力し、相対順位、スコア差、同率1位アラート、precision / recall / F1を再現可能なJSONとして出力します。

## 実行

```bash
go run ./cmd/lumi-exporter -input testdata/evaluation.json
```

`-pretty=false`を指定すると、1行JSONとして出力します。判定結果には次の情報を含みます。

- `matched` / `ambiguous` / `unmatched`
- `top_score`、`second_score`、`score_margin`
- title、date、venue、performersごとの特徴量とcoverage
- 同率1位やマージン不足などのレビューアラート
- 自動統合のprecision、recall、F1、top-1 accuracy

## pull型Webサービス

登録済みソースをpullして、共通データを画面で確認するには次を実行します。

```bash
go run ./cmd/lumi-exporter -serve
open http://127.0.0.1:8081/
```

登録内容は [`config/sources.json`](config/sources.json) で管理します。現在の初期登録は次の3つです。

| サービス | adapter | endpoint |
|---|---|---|
| `lumitlink` | `entity.v1` | 雨宮サクの共通エンティティ |
| `lumitree` | `lumitree_calendar` | `ilife_official`のイベント一覧 |
| `lumidive` | `lumidive_event` | `plkt1022`のイベント |

APIは次の通りです。

- `GET /api/v1/sources`: 登録内容
- `GET /api/v1/catalog`: 全登録先をpullした結果
- `GET /api/v1/sources/{id}/pull`: 一つの登録先だけpullした結果
- `GET /openapi.yaml`: WebサービスのAPI定義

画面には、HTTPステータス、取得時間、endpoint、`lumi-spec`の正規化データ、生JSONを表示します。画面操作が実際のpullへ到達したことは、次のログで確認できます。

```bash
tail -f /tmp/lumi-exporter.log
# Kubernetesで運用する場合
kubectl logs -f deployment/lumi-exporter -n lumi-exporter
```

初期PoCの完了条件は、登録済みの`lumitlink`、`lumitree`、`lumidive`から少なくとも1件ずつHTTP取得でき、各結果が共通契約または明示したadapterで描画され、画面から生JSONと取得メタデータを追跡できることです。認証、永続DB、定期poll、外部公開用のSSRF対策は次段階とします。

## 設計境界

- 候補の取得・正規化は入力アダプターの責務であり、このPoCは候補集合のランキングに集中します。
- `lumi-spec`を唯一の共有依存とし、`lumitree`や`lumidive`の実装を直接importしません。
- 同率1位やスコア差不足は成功扱いにせず、`ambiguous`としてレビューへ送ります。
- しきい値は実データのラベル付きfixtureで評価してから変更します。

## 検証

```bash
./scripts/verify-all.sh
```

出力JSONは、画面の見た目に依存せず、判定理由と元候補を確認するための一次データです。実運用でUIを追加する場合も、このJSONとバックエンドログを確認できる導線を残します。

## 関連仕様

- [`lumi-spec`](https://github.com/AobaIwaki123/lumi-spec)
- [`event-candidate.v1`](https://raw.githubusercontent.com/AobaIwaki123/lumi-spec/main/schema/event-candidate.v1.json)
- [`match-decision.v1`](https://raw.githubusercontent.com/AobaIwaki123/lumi-spec/main/schema/match-decision.v1.json)
