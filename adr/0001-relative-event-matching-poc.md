# ADR 0001: 相対評価によるイベント名寄せPoC

## ステータス

採用予定

## 背景

`lumitree`、`lumidive`など複数ソースのイベント名は、空白、大小文字、記号、表記揺れによって完全一致しない。一方、同一候補集合の中に極端に似たイベントが多数あるとは限らないため、候補集合内の相対順位を使う余地がある。

ただし、相対順位だけで自動統合すると、候補追加や正規化ルールの変更で誤統合が起こる。1位が同点になるケースは「同一イベント」とは解釈せず、判定器が区別できていない信号としてレビューへ戻す。

## 決定

`lumi-spec`の`event-candidate.v1`を入力にし、候補ごとに次の特徴量を計算する。

| 特徴量 | 初期の扱い | 目的 |
|---|---|---|
| title | Unicode NFKC、空白・記号除去、完全一致・包含・Dice係数 | 表記揺れを吸収する |
| date | 同日=1、前後1日=0.5、それ以外=0 | 同名イベントの分離 |
| venue | titleと同じ文字列類似度 | 会場の識別 |
| performers | 正規化した集合のJaccard係数 | 出演者情報の利用 |

初期重みは`title=0.45`、`date=0.30`、`venue=0.15`、`performers=0.10`とする。候補を総合スコアで順位付けし、以下のルールで判定する。

- 候補なし、または`top_score`が最低値未満なら`unmatched`
- 1位と2位が同率なら`ambiguous`かつ`top_score_tie`アラート
- 1位と2位の差が`min_margin`未満なら`ambiguous`かつ`score_margin_below_minimum`アラート
- それ以外で最低スコア以上なら`matched`

`ambiguous`では候補ID、特徴量、スコア、元の`source_ref`を保持し、自動統合しない。評価fixtureでfalse mergeが0件であることを確認してから、実データを増やして閾値を再評価する。

## 捨てたものとリスク

初期PoCでは機械学習モデル、外部のレコードリンケージ基盤、候補生成ロジックを採用しない。少量のデータで説明可能性とレビュー可能性を優先できる反面、イベント数や表記の多様性が増えると特徴量と重みの再設計が必要になる。

`local_date`は入力アダプターがタイムゾーンを考慮して生成する。名寄せ器が暗黙に変換しないことで日付境界の誤判定を追跡できるが、入力側の品質に依存する。

参考:

- [JSON Schema 2020-12](https://json-schema.org/draft/2020-12/json-schema-core)
- [Unicode Standard Annex #15: Unicode Normalization Forms](https://unicode.org/reports/tr15/)
- [Fellegi and Sunter, A Theory for Record Linkage](https://doi.org/10.1080/01621459.1969.10501049)
