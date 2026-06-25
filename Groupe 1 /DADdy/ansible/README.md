# Ansible K3s Single Node

Ce dossier contient un playbook pour installer un cluster K3s single node sur une VM Debian 13 a partir de son IP.

Il installe aussi les services de base de `.kubernetes/platform/bootstrap/shared`:

- Argo CD
- namespaces plateforme
- Sealed Secrets pour les credentials AWS Route53 et external-dns
- applications Argo CD `cert-manager` et `external-dns`
- `ClusterIssuer` Let's Encrypt staging et production

## Prerequis

- Ansible installe sur ta machine locale.
- Acces SSH a la VM.
- Un utilisateur SSH avec droits `sudo`.
- La VM doit pouvoir acceder a Internet pour installer K3s, Argo CD et les charts declares dans `.kubernetes`.

## Lancement rapide avec une IP

Depuis la racine du repo:

```bash
ansible-playbook -i "192.0.2.10," -u debian ansible/playbooks/k8s-single-node.yml
```

Avec une cle SSH specifique:

```bash
ansible-playbook -i "192.0.2.10," -u debian --private-key ~/.ssh/id_ed25519 ansible/playbooks/k8s-single-node.yml
```

Si ton utilisateur demande un mot de passe sudo:

```bash
ansible-playbook -i "192.0.2.10," -u debian --ask-become-pass ansible/playbooks/k8s-single-node.yml
```

## Lancement avec l'inventaire exemple

Edite `ansible/inventory.example.ini`, puis lance:

```bash
ansible-playbook -i ansible/inventory.example.ini ansible/playbooks/k8s-single-node.yml
```

Pre-check sans appliquer de changements:

```bash
ansible-playbook -i ansible/inventory.example.ini ansible/playbooks/k8s-single-node.yml --check
```

## Variables utiles

Tu peux surcharger les valeurs principales avec `-e`:

```bash
ansible-playbook -i "192.0.2.10," -u debian ansible/playbooks/k8s-single-node.yml \
  -e k3s_channel=stable \
  -e k3s_api_advertise_address=192.0.2.10
```

- `k3s_channel`: canal K3s a installer. Par defaut `stable`.
- `k3s_api_advertise_address`: IP exposee par l'API server. Par defaut, l'IP de l'inventaire.
- `k3s_tls_sans`: IPs ou DNS autorises dans le certificat de l'API server. Par defaut, l'IP de la VM est incluse.
- `k3s_cluster_cidr`: CIDR pods. Par defaut `10.42.0.0/16`, valeur K3s standard.
- `k3s_service_cidr`: CIDR services. Par defaut `10.43.0.0/16`, valeur K3s standard.
- `k3s_local_kubeconfig_path`: chemin local ou copier le kubeconfig K3s. Par defaut `~/.kube/config`.
- `k3s_disable_components`: composants K3s a desactiver, par exemple `["traefik"]`. Par defaut vide, donc Traefik reste actif.
- `k3s_extra_server_args`: arguments supplementaires passes a `k3s server`.
- `kubernetes_apply_platform_bootstrap`: applique les services de base `.kubernetes/platform/bootstrap/shared`. Par defaut `true`.
- `kubernetes_apply_argocd_root_app`: applique aussi `.kubernetes/argocd/root-application.yaml`. Par defaut `false` pour eviter de deployer les apps applicatives sans validation explicite.
- `kubernetes_remote_manifests_dir`: dossier distant ou `.kubernetes` est copie. Par defaut `/opt/daddy/.kubernetes`.

Pour ajouter un DNS ou une autre IP au certificat de l'API server:

```bash
ansible-playbook -i "192.0.2.10," -u debian ansible/playbooks/k8s-single-node.yml \
  -e '{"k3s_tls_sans":["192.0.2.10","k8s.example.com"]}'
```

## Apres installation

Le playbook copie le kubeconfig dans `/home/<user>/.kube/config` sur la VM et dans `~/.kube/config` sur ta machine locale. Pour tester depuis la VM ou depuis ta machine locale:

```bash
kubectl get nodes -o wide
kubectl get pods -A
```
