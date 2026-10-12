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
