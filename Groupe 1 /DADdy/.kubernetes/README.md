# Kubernetes GitOps Layout

This repository can bootstrap one shared cluster platform and two application environments from a single `.kubernetes/` folder.

## Structure

```text
.kubernetes/
├── argocd/
│   ├── apps/
│   │   ├── daddy-dev.yaml
│   │   ├── daddy-prod.yaml
│   │   ├── kustomization.yaml
│   │   └── platform.yaml
│   ├── project.yaml
│   └── root-application.yaml
├── apps/
│   └── daddy/
│       ├── base/
│       └── overlays/
│           ├── dev/
│           └── prod/
└── platform/
    └── bootstrap/
        └── shared/
```

## What Is Included

- Argo CD app-of-apps bootstrap
- K3s default Traefik ingress controller
- Sealed Secrets installation by Argo CD
- `cert-manager` installation by Argo CD
- `external-dns` installation by Argo CD
- Route53-backed DNS-01 `ClusterIssuer` for Let's Encrypt staging in `dev`
- Route53-backed DNS-01 `ClusterIssuer` for Let's Encrypt production in `prod`
- separate `daddy-dev` and `daddy` application overlays
- TLS `Certificate` objects per environment

## What You Still Need To Replace

- `repoURL` values in the Argo CD `Application` manifests
- placeholder image registries and tags in the app overlays
- placeholder domain names in ingress and certificate manifests
- placeholder email addresses in the `ClusterIssuer` manifests
- Route53 hosted zone ID in the `ClusterIssuer` manifests
- AWS credentials by generating new SealedSecret manifests in `platform/bootstrap/shared`
- `AWS_DEFAULT_REGION` in `platform/bootstrap/shared/external-dns.yaml`
- bootstrap secrets in `apps/daddy/overlays/*/secrets.yaml`

## Bootstrap

Apply Argo CD once, then let it reconcile everything else:

1. Apply [`argocd/project.yaml`](/Users/enzo-cadiere/DADdy/.kubernetes/argocd/project.yaml:1)
2. Apply [`argocd/root-application.yaml`](/Users/enzo-cadiere/DADdy/.kubernetes/argocd/root-application.yaml:1)

The root app reconciles the three child applications from this same repo:

- `platform`
- `daddy-dev`, from the `develop` branch
- `daddy-prod`, from the `main` branch

## Notes

- `cert-manager` is installed from the official OCI Helm chart. The cert-manager docs currently recommend the OCI chart and show `v1.20.2` as the latest chart example.
- Sealed Secrets is installed from the Bitnami Labs Helm chart and stores AWS credentials as encrypted `SealedSecret` resources.
- `external-dns` is installed from the official Helm chart and configured for AWS Route53 with `Ingress` as the source.
- This layout assumes a default K3s installation, so it uses the built-in Traefik ingress controller instead of installing `ingress-nginx`.
- ACME is configured with DNS-01 against AWS Route53, so certificate issuance does not depend on HTTP challenge routing through the ingress controller.
- The cluster is shared. Only the dev namespace ends with `-dev`; prod uses the base namespace `daddy`.
- `prod` uses the Let's Encrypt production ACME endpoint. `dev` uses the staging endpoint to avoid rate limits.
