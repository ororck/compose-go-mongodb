# Guide — TP Docker Compose (Go + MongoDB)

Guide d'autonomie pour atteindre le résultat final. Chaque section pose une **question à résoudre** et donne les **pistes** pour y répondre toi-même, pas la solution complète.

---

## Objectif final

Écrire un `compose.yml` qui, en **une seule commande**, démarre :
- l'application Go (exposée sur le port **5000** de la machine)
- la base MongoDB

Avec toutes les contraintes des critères de performance (voir tout en bas).

---

## Étape 0 — Comprendre l'existant

Avant d'écrire quoi que ce soit, relis ces fichiers du repo et réponds à ces questions :

- `main.go` : quelles **variables d'environnement** l'app attend-elle pour se connecter à Mongo ? (cherche `os.LookupEnv`)
- `main.go` : sur quel **port** l'app écoute-t-elle par défaut ? (Gin lit la variable `PORT`, défaut `8080`)
- `Dockerfile` : c'est un build **multi-stage**. Quelle image sert à builder, laquelle sert au runtime ?
- Quelle image publique MongoDB est utilisée ? (voir l'ancien `Makefile` : `mongo:7.0`)

> Checkpoint : tu dois savoir citer les 2 variables Mongo attendues et le port de l'app.

---

## Étape 1 — Les réseaux Docker (partie évaluée à l'oral)

Le TP est **évalué** sur ta capacité à expliquer les réseaux. Ne saute pas cette partie.

Recherche et rédige (dans un fichier `NOTES.md` ou le README) des réponses courtes à :

1. Quels sont les **réseaux par défaut** créés à l'installation de Docker ? (`docker network ls`)
2. Sur quel réseau atterrit un conteneur lancé avec un simple `docker run` sans `--network` ?
3. Différence entre :
   - **bridge** (le défaut)
   - **host**
   - un **réseau bridge custom** (celui que Compose crée)
4. Pourquoi deux conteneurs sur le **bridge par défaut** ne peuvent pas se joindre par leur **nom**, alors que sur un **bridge custom** oui ? (mot-clé : résolution DNS intégrée)
5. Comment l'isolation est réalisée au niveau de l'OS ? (mots-clés à creuser : **network namespaces**, **veth pair**, **iptables/nftables**)

Commandes utiles pour observer :
```bash
docker network ls
docker network inspect bridge
docker network create matest && docker network inspect matest
```

> Checkpoint : tu sais expliquer host vs bridge vs custom, et pourquoi le nom de conteneur résout sur un réseau custom.

---

## Étape 2 — Un `compose.yml` minimal qui marche

Commence simple, tu ajouteras les contraintes après. Objectif de cette étape : les 2 services démarrent et communiquent.

Structure à construire toi-même :

```yaml
services:
  app:
    build: .          # construit l'image depuis le Dockerfile local
    ports:
      - "?:?"          # QUESTION : quel mapping pour exposer sur 5000 ?
    environment:
      MONGODB_HOST: ?  # QUESTION : quelle valeur ? (indice : le NOM du service mongo)
      MONGODB_PORT: ?
      PORT: ?
    depends_on:
      - ?

  mongo:
    image: ?           # QUESTION : quelle image publique ?
    # ...
```

Questions à résoudre pour cette étape :
- **Port** : l'app écoute sur `8080` dans le conteneur. Tu veux y accéder via `5000` sur ta machine. Quel est le bon format `"hôte:conteneur"` ?
- **MONGODB_HOST** : sur un réseau Compose, un service joint un autre par son **nom de service**. Quel nom mettre ?
- Compose crée-t-il un réseau tout seul, ou dois-tu le déclarer ? (teste sans rien déclarer d'abord)

Teste :
```bash
docker compose up --build
# autre terminal :
curl localhost:5000/healthz
curl localhost:5000/
curl localhost:5000/logs
```

> Checkpoint : `curl localhost:5000/healthz` renvoie `{"status":"healthy"}`.

---

## Étape 3 — Tâche de fond + redémarrage automatique

Deux critères d'un coup :

- **Tourner en tâche de fond** → quel flag sur `docker compose up` ? (indice : `-d`)
- **Redémarrage auto en cas d'échec** → quelle clé ajouter à chaque service ? Cherche `restart:` et compare les valeurs `no`, `on-failure`, `always`, `unless-stopped`. Laquelle correspond à « redémarré en cas d'échec » ?

> Checkpoint : `docker compose ps` montre les 2 services `Up`, et tuer un conteneur le fait repartir.

---

## Étape 4 — Persistance de MongoDB (volume)

Sans volume, les données Mongo disparaissent au `docker compose down`.

Questions :
- Dans quel dossier MongoDB stocke-t-il ses données à l'intérieur du conteneur ? (cherche « mongodb data directory » — c'est `/data/db`)
- Comment déclarer un **named volume** dans Compose, et le monter sur ce dossier ?

Tu auras besoin :
- d'une section `volumes:` sur le service `mongo`
- d'une section `volumes:` au niveau **racine** du fichier pour déclarer le volume nommé

Test de validation :
```bash
docker compose up -d
curl localhost:5000/        # crée une entrée
docker compose down         # SANS -v
docker compose up -d
curl localhost:5000/logs    # l'entrée doit toujours être là
```

> Checkpoint : les logs survivent à un `down` + `up`.

---

## Étape 5 — Hot reload de l'app Go (`develop: watch`)

Critère : l'app Go doit **rebuild/redémarrer automatiquement** dès qu'un fichier `.go` change.

C'est la fonctionnalité **`develop.watch`** de Compose (Compose >= 2.22).

Questions à résoudre :
- Quelle est la syntaxe de la section `develop:` → `watch:` d'un service ?
- Quelle **action** choisir parmi `sync`, `rebuild`, `sync+restart` ? (indice : ton app est compilée, un simple sync du fichier source ne suffit pas — il faut re-builder l'image)
- Quel `path` surveiller, et comment cibler les fichiers `.go` ?

Pour activer la surveillance :
```bash
docker compose watch
```

Test : modifie un `.go` (ex. change un message dans un handler), sauvegarde, observe le rebuild automatique.

> Checkpoint : éditer un `.go` déclenche un rebuild sans commande manuelle.

---

## Étape 6 — Secrets Docker + exclusion Git

Critère : les variables sensibles (ex. mot de passe utilisateur Mongo) passent par des **secrets Docker**, et ne sont **jamais commit**.

Étapes à concevoir :
1. MongoDB peut créer un utilisateur root via `MONGO_INITDB_ROOT_USERNAME` et `MONGO_INITDB_ROOT_PASSWORD`. La variante `_FILE` (`MONGO_INITDB_ROOT_PASSWORD_FILE`) permet de **lire la valeur depuis un fichier** → c'est ce qui se marie avec les secrets.
2. Déclare un **secret** dans Compose (section `secrets:` racine + `secrets:` sur le service) pointant vers un fichier local, ex. `secrets/mongo_password.txt`.
3. Ajoute ce fichier de secret à `.gitignore`.

Questions :
- Où Docker monte-t-il un secret dans le conteneur ? (indice : `/run/secrets/<nom>`)
- Faut-il alors pointer `MONGO_INITDB_ROOT_PASSWORD_FILE` vers ce chemin ?
- Si Mongo a maintenant un user/password, ta **connection string** côté app doit-elle changer ? (regarde comment `main.go` construit l'URI — tu devras peut-être passer user/pass à l'app aussi)

> Checkpoint : `git status` ne montre jamais le fichier de mot de passe, et Mongo démarre avec l'auth activée.

---

## Étape 7 — Ordre de démarrage (app APRÈS la DB)

Critère : l'app ne démarre **qu'une fois la DB réellement prête**.

Attention au piège : `depends_on` **simple** attend que le conteneur soit *lancé*, pas que Mongo soit *prêt à accepter des connexions*.

Questions :
- Quelle est la forme **longue** de `depends_on` avec `condition:` ?
- Quelle condition attend la **santé** du service ? (indice : `service_healthy`)
- Il faut donc définir un **`healthcheck:`** sur le service Mongo. Quelle commande teste que Mongo répond ? (cherche `mongosh --eval "db.adminCommand('ping')"`)

> Checkpoint : au démarrage à froid, l'app ne log plus d'erreur « failed to ping MongoDB ».

---

## Étape 8 — Livrable Git

- Initialise le repo (ou réutilise le repo cloné).
- Vérifie que `.gitignore` exclut : le dossier/fichier de secrets, et tout binaire local (`main`).
- Commit `compose.yml`, `.gitignore`, tes notes réseau, et pousse sur la plateforme de ton choix (GitHub/GitLab).

Vérification finale :
```bash
git status          # aucun secret listé
docker compose down -v && docker compose up -d --build
curl localhost:5000/healthz
```

---

## Récapitulatif des critères (à cocher)

- [ ] Une seule commande démarre toute l'infra
- [ ] App accessible sur le port **5000**
- [ ] Les 2 conteneurs tournent en **tâche de fond**
- [ ] **Restart automatique** en cas d'échec
- [ ] Données Mongo **persistées** dans un volume
- [ ] App Go **rebuild/redémarre** à toute modif `.go`
- [ ] Mot de passe via **secret Docker**, exclu de Git
- [ ] App démarre **après** que la DB soit prête (healthcheck + depends_on condition)
- [ ] Repo Git livré
- [ ] Tu sais expliquer **host / bridge / réseau custom** à l'oral

---

## Antisèche des mots-clés Compose à chercher

`services` · `build` · `image` · `ports` · `environment` · `depends_on` (+ `condition: service_healthy`) · `restart` · `volumes` (service + racine) · `develop` → `watch` (`action: rebuild`) · `secrets` (service + racine) · `healthcheck` · commandes : `docker compose up -d --build`, `docker compose watch`, `docker compose ps`, `docker compose logs -f`, `docker compose down [-v]`