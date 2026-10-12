# lumi-exporter イベント名寄せPoC ディスカッション & 進捗管理 (discussion.md)

このドキュメントでは、lumi-exporter イベント名寄せPoC に関する目的、ロードマップ、議論の経緯、進捗を記録・管理します。

---

## 目的・ゴール

- `lumitree`、`lumidive`などのイベント候補を`lumi-spec`へ正規化した入力として受け取る。
- 候補集合内の相対評価でイベントを順位付けし、同率1位やマージン不足をレビューアラートにする。
- ラベル付きfixtureでprecision、recall、F1、false merge、ambiguous率を測定する。
- 元イベントのID、URL、原文を残し、判定ルールを変更して再評価できるようにする。

---

## ロードマップ / タスク

| Step | 項目 | 内容 | 状況 |
| :--- | :--- | :--- | :--- |
| **Step 1** | **調査・要件定義** | 相対評価、同率アラート、評価指標を定義 | 完了 |
| **Step 2** | **設計・方針決定** | `lumi-spec`依存、特徴量、保留条件をADR化 | 完了 |
| **Step 3** | **実装・スクリプト化** | Go matcher、fixture、評価CLI、検証スクリプト | 完了 |
| **Step 4** | **pullカタログ** | 登録済みlumiサービスのpull、共通化、簡易UI | 進行中 |
| **Step 5** | **実データ評価** | lumitree/lumidiveのラベル付きfixtureを追加 | 未着手 |

---

## 決定事項 (ADR一覧)

- [ADR 0001: 相対評価によるイベント名寄せPoC](adr/0001-relative-event-matching-poc.md)
- [ADR 0002: 登録済みlumiサービスをpullする薄いWebサービス](adr/0002-pull-catalog-service.md)

---

## 直近のネクストアクション

- [ ] `lumi-spec`のリリースタグへ依存を固定する
- [ ] lumitree/lumidiveの実データを匿名化した評価fixtureへ追加する
- [ ] false mergeを最優先にprecisionとrecallを再評価する
- [ ] 同率1位アラートをコードレビューの入力として運用する
- [ ] `lumibot`を将来候補として検討する（現状は未接続）
