# compose-go-mongodb
 
Infrastructure conteneurisée avec **Docker Compose** : une API web en **Go** (framework Gin) connectée à une base de données **MongoDB**, avec persistance, gestion des secrets, health checks et hot reload en développement.
 
Ce dépôt couvre deux briefs successifs :
1. **Brief 1** — Orchestration Compose de l'application Go + MongoDB.
2. **Brief 2** — Mise à l'échelle de l'application et répartition de charge via un load balancer.
---
 
## Architecture
 
L'application expose trois routes HTTP :
 
| Méthode | Route       | Description                                              |
|---------|-------------|---------------------------------------------------------|
| `GET`   | `/`         | Insère un document horodaté dans MongoDB et le renvoie. |
| `GET`   | `/healthz`  | Renvoie l'état de santé de l'application.                |
| `GET`   | `/logs`     | Renvoie tous les documents stockés en base.             |
 
---
 
## Prérequis
 
- Docker Engine et Docker Compose v2 (`docker compose version`)
- Un fichier de secret pour le mot de passe MongoDB (voir ci-dessous)
---
 
## Démarrage rapide
 
Avant le premier lancement, créer le secret contenant le mot de passe MongoDB :
 
```bash
mkdir -p secrets
echo "monMotDePasseSolide" > secrets/mongo_password.txt
```
 
Puis démarrer l'infrastructure :
 
```bash
docker compose up -d --build
```
 
Vérifier :
 
```bash
curl localhost:5000/healthz    # {"status":"healthy"}
curl localhost:5000/           # crée une entrée
curl localhost:5000/logs       # liste les entrées
```
 
Arrêter (sans supprimer les données) :
 
```bash
docker compose down
```
 
Arrêter et supprimer le volume de données :
 
```bash
docker compose down -v
```
 
---
 
## Brief 1 — Compose Go + MongoDB
 
### Fonctionnalités mises en place
 
- **Une seule commande** démarre toute l'infrastructure (application + base de données).
- L'application est exposée sur le port **5000** de la machine hôte.
- Les deux conteneurs tournent en **tâche de fond** et sont **redémarrés automatiquement** en cas d'échec (`restart`).
- Les données MongoDB sont **persistées** dans un volume nommé (`mongo_data`), montée sur `/data/db`.
- L'application Go est **reconstruite et redémarrée automatiquement** à chaque modification d'un fichier `.go`, grâce à la fonctionnalité `develop.watch` de Compose.
- Le **mot de passe MongoDB** est géré via un **secret Docker** (`secrets/mongo_password.txt`), exclu de tout commit via `.gitignore`.
- L'application ne démarre **qu'après** que la base de données soit réellement prête, grâce à un **health check** sur MongoDB et à `depends_on: condition: service_healthy`.
### Hot reload en développement
 
Pour activer la surveillance des fichiers `.go` et le rebuild automatique :
 
```bash
docker compose watch
```
 
Toute modification d'un fichier source déclenche une reconstruction de l'image et le redémarrage du service `app`.
 
### Gestion du secret
 
Le mot de passe n'est jamais écrit en clair dans `compose.yml`, ni exposé en variable d'environnement (invisible dans `docker inspect`).
 
- Côté MongoDB : le mot de passe est lu depuis un fichier via `MONGO_INITDB_ROOT_PASSWORD_FILE`.
- Côté application : le mot de passe est lu depuis le fichier secret monté dans `/run/secrets/`, puis fourni au driver MongoDB via l'objet d'authentification `options.Credential` (les identifiants ne transitent pas par la chaîne de connexion).
Le dossier `secrets/` est listé dans `.gitignore`.
 
### Les réseaux Docker
 
Docker crée par défaut trois réseaux : `bridge`, `host` et `none`.
 
- **bridge (par défaut)** : un conteneur lancé avec `docker run` sans option `--network` rejoint le réseau `bridge` par défaut. Les conteneurs y communiquent par adresse IP, mais **pas par leur nom** (pas de résolution DNS intégrée).
- **host** : le conteneur partage directement la pile réseau de l'hôte, sans isolation ni translation de ports.
- **réseau bridge personnalisé** : c'est ce que crée automatiquement Docker Compose. Contrairement au bridge par défaut, il fournit une **résolution DNS interne** : un service peut joindre un autre par son **nom de service** (ici l'application joint MongoDB via `mongo`). C'est pourquoi la variable `MONGODB_HOST` vaut `mongo`.
L'isolation est réalisée au niveau du système d'exploitation via les **network namespaces** du noyau Linux (chaque conteneur possède sa propre pile réseau), reliés par des **paires d'interfaces virtuelles (veth)** au bridge, le filtrage et la translation d'adresses étant gérés par **iptables/nftables**.
 
---
 
## Brief 2 — Mise à l'échelle et load balancing
 
> Section en cours. Objectif : mettre l'application à l'échelle (plusieurs réplicas) et placer un load balancer devant elle.
 
### Objectif
 
Passer d'un unique conteneur applicatif à **plusieurs réplicas** pour absorber davantage de trafic, et ajouter un **load balancer** (image `dockercloud/haproxy`) agissant comme **reverse proxy** : ce n'est plus l'application qui est exposée directement sur le port 5000, mais le load balancer placé devant les réplicas.
 
### Démarche
 
1. Tenter une mise à l'échelle de l'application via Compose (`docker compose up --scale app=3`, ou la clé `deploy.replicas`).
2. Retirer l'exposition directe du port de l'application pour autoriser plusieurs réplicas (deux conteneurs ne peuvent pas publier le même port hôte).
3. Constater que l'application n'est plus joignable depuis le web.
4. Ajouter un service **load balancer** dans `compose.yml`.
5. Le configurer pour rediriger le trafic vers les réplicas de l'application Go.
6. Contacter à nouveau l'application, cette fois via le load balancer.
7. Répéter les requêtes en observant les logs des réplicas pour vérifier la répartition.
8. Constater que le trafic est distribué équitablement entre les réplicas.
### Vérification de la répartition
 
```bash
# Envoyer plusieurs requêtes
for i in $(seq 1 10); do curl -s localhost/ > /dev/null; done
 
# Observer que les requêtes se répartissent entre les réplicas
docker compose logs app
```
 
### Bonus
 
- Chiffrement du trafic vers le load balancer via **TLS** (application accessible uniquement en `https://`).
- Séparation réseau : le load balancer tourne dans un réseau **front-end**, tandis que l'application et la base de données partagent un réseau **back-end** isolé.
---
 
## Structure du dépôt
 
```
.
├── compose.yml            # orchestration des services
├── Dockerfile             # build multi-stage de l'application Go
├── .dockerignore
├── .gitignore             # exclut notamment secrets/
├── main.go                # point d'entrée, connexion MongoDB
├── handlers.go            # handlers HTTP
├── go.mod / go.sum
├── Makefile               # cibles utilitaires
└── secrets/               # NON versionné — contient mongo_password.txt
```
 
---
 
## Critères couverts
 
**Brief 1**
- [x] Une seule commande démarre l'infrastructure
- [x] Application accessible sur le port 5000
- [x] Conteneurs en tâche de fond avec redémarrage automatique
- [x] Données MongoDB persistées dans un volume
- [x] Rebuild/redémarrage automatique de l'app à chaque modification `.go`
- [x] Mot de passe via secret Docker, exclu de Git
- [x] Application démarrée uniquement après que la base soit prête

**Brief 2**
- [ ] L'application reste accessible
- [ ] Le load balancer reçoit le trafic
- [ ] Le trafic est distribué entre les réplicas (vérifié via les logs)
