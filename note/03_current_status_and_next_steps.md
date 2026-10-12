# 03: lumi-exporterの現在地と次の検証

## 1. 背景と現象の観察

`lumi-exporter`は、`lumi-spec`の共通契約を利用して、`lumitlink`、`lumitree`、`lumidive`の公開APIをpullし、正規化結果と上流の生JSONを同じ画面で確認する薄いWebサービスとして実装した。

2026-10-12時点で、公開ホスト `lumi-exporter.aooba.net` のIngress経路まで反映されている。通常の名前解決を使ったこの実行環境の`curl`では一時的に名前解決エラーが発生したが、DNSレコードはCloudflareのIPへ解決され、ホスト名を明示したHTTPS接続では応答を確認できた。

## 2. 構造分析 / 現在の状態

| 対象 | 状態 | 確認結果 |
|---|---|---|
| `lumi-spec` | 完了 | `v0.1.1`を公開。Entity/Event/Feed/Match契約を提供 |
| `lumitlink` | 稼働 | `/api/v1/entities/amamiyasaku`が`lumi.entity.v1`を返す |
| `lumi-exporter` CI | 完了 | Go CIとGHCRイメージ生成が成功 |
| `lumi-exporter` Ingress | 稼働 | `/healthz`がHTTP 200、`{"service":"lumi-exporter","status":"ok"}` |
| `lumitlink` pull | 成功 | HTTP 200、Entity名「雨宮 サク」 |
| `lumitree` pull | 成功 | HTTP 200、正規化イベント12件 |
| `lumidive` pull | 成功 | HTTP 200、正規化イベント1件 |
| ローカルKubernetes API確認 | 未確認 | この実行環境から`192.168.11.221:6443`へ到達できない |

公開catalogの実データ確認には、CloudflareのエッジIPを指定したHTTPS接続を使用した。これはアプリケーション障害ではなく、実行環境の通常DNSリゾルバと公開DNSの一時的な差異を切り分けるためである。

## 3. アクションプロトコル / 運用ルール

### 完了済み

- `lumi-spec`をGo moduleの共有依存として固定する。
- `lumitlink`に`/api/v1/entities/{username}`を追加する。
- `lumi-exporter`に登録ソース、pull adapter、catalog API、簡易UIを追加する。
- UIでHTTPステータス、取得時間、endpoint、正規化データ、生JSONを確認できるようにする。
- Kubernetesでは数値UID `65532`を使い、`runAsNonRoot`の検証問題を避ける。
- Argo CD ApplicationとCloudflare Tunnel IngressをGit管理する。

### 次に行うこと

1. 実データから匿名化した正解ラベル付きfixtureを作り、イベント名寄せのprecision、recall、false merge、ambiguous率を再評価する。
2. 同率1位・マージン不足アラートをレビュー運用に接続する。
3. `lumitree`と`lumidive`が将来的に`lumi-spec`のfeed envelopeを直接返せるよう、サービス側の契約追加を検討する。
4. 外部公開を継続する場合は、認証、endpoint許可リスト、SSRF防御、定期pull、永続化を追加する。
5. DNS解決に問題があるクライアントでは、ブラウザ再起動・ネットワーク再接続・端末再起動の順に確認する。

Organizationへの移行は現時点では必須ではない。一覧性だけが目的であれば、個人プロフィールREADME、GitHub Project、サービス一覧ファイルで代替する。共同開発、権限管理、独立ブランド化が必要になった時点で移行を再検討する。

---

## 参考文献 (Authoritative References)

- [lumi-spec](https://github.com/AobaIwaki123/lumi-spec)
- [lumi-exporter](https://github.com/AobaIwaki123/lumi-exporter)
- [lumitlink](https://github.com/AobaIwaki123/lumitlink)
- [Kubernetes Ingress](https://kubernetes.io/docs/concepts/services-networking/ingress/)
- [STRRL Cloudflare Tunnel Ingress Controller](https://github.com/STRRL/cloudflare-tunnel-ingress-controller)
- [OWASP SSRF Prevention Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Server_Side_Request_Forgery_Prevention_Cheat_Sheet.html)
