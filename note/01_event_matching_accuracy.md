# イベント名寄せ精度の初期評価

## 目的

相対評価を採用したとき、同率1位を自動統合せずアラートへ送れるか、またラベル付きケースで自動統合の誤りを測定できるかを確認する。

## 評価方法

`testdata/evaluation.json`に、共通の`lumi.event.candidate.v1`形式で6ケースを用意した。

| ケース | 期待結果 | 見ている挙動 |
|---|---|---|
| exact-with-notation-variance | matched | 空白・記号の表記揺れ |
| date-breaks-same-title | matched | 同名イベントの日付分離 |
| top-score-tie | ambiguous | 同率1位アラート |
| low-score-unmatched | unmatched | 最低スコア未満 |
| margin-review-with-shared-performer | ambiguous | 相対差不足 |
| missing-optional-fields | matched | 任意項目欠落時の挙動 |

CLIの出力には、ケースごとの判定に加えて、次の集計値を含める。

- 自動統合のprecision / recall / F1
- top-1 accuracy
- false merge数、missed match数
- `ambiguous`率とアラート理由別件数

初期fixtureを`DefaultConfig`で実行した結果は次の通りだった。

| 指標 | 結果 |
|---|---:|
| ケース数 | 6 |
| 自動統合候補 | 3 |
| correct auto matches | 3 |
| false merge | 0 |
| missed match | 0 |
| auto-match precision | 1.0 |
| auto-match recall | 1.0 |
| auto-match F1 | 1.0 |
| top-1 accuracy | 1.0 |
| ambiguous率 | 0.3333 |
| 同率1位アラート | 1 |

これは人工的なfixtureに対する再現性確認であり、実データの品質保証ではない。特に`precision=1.0`をそのまま本番精度と解釈せず、実データを追加してfalse mergeの有無を最優先に確認する。

このfixtureは実データの精度を代表するものではない。次の段階では`lumitree`と`lumidive`から匿名化した実例を収集し、人手で同一イベントの正解ラベルを付けてfixtureへ追加する。

## 再現コマンド

```bash
./scripts/verify-all.sh
go run ./cmd/lumi-exporter -input testdata/evaluation.json
```

## 解釈上の注意

- `ambiguous`は失敗ではなく、自動統合を保留した件数である。
- 同率1位は判定器の識別力不足を示すため、アラート件数をゼロにすることを目標にしない。
- 閾値を緩めてmatched数だけを増やすと、false mergeが増える可能性がある。precisionとrecallを同時に見る。
- 元の`source_ref`を保持するため、判定ロジックを変更して再評価できる。

参考:

- [JSON Schema 2020-12](https://json-schema.org/draft/2020-12/json-schema-core)
- [Unicode Standard Annex #15: Unicode Normalization Forms](https://unicode.org/reports/tr15/)
- [scikit-learn classification metrics](https://scikit-learn.org/stable/api/sklearn.metrics.html)
- [Fellegi and Sunter, A Theory for Record Linkage](https://doi.org/10.1080/01621459.1969.10501049)
