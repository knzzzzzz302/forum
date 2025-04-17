Bonjour ou bonsoir on vous présente notre Projet FORUM, un projet qui a pour but de construire un site forum avec des fonctionnalité comme la publication de post,likes,etc

V

    Inscription et connexion classiques

    Authentification à deux facteurs

    Connexion avec Google ou GitHub

    Protection brute force + limitation des requêtes

    Créer, éditer et commenter des posts

    Like/dislike sur les posts

    Upload d’images

    Recherche par mot-clé et filtres

    Profil perso avec stats

Installation rapide

Ce qu’il te faut :

    Go 1.23.0 ou +

    Git

Pour lancer le serveur :

git clone https://github.com/username/forum-sekkay.git
cd forum-sekkay
go mod download
go build -o app main.go
./app

En gros après t’ouvres http://localhost:3030/ et c’est bon.

HTTPS (optionnel) :

./app --https

Avec Docker :

docker build -t forum-sekkay .
docker run -p 3030:3030 -it forum-sekkay

( !! LA CONNEXION GOOGLE ET GITHUB ne marchent pas avec DOCKER on a tout essayer on s'est rendu compte que c'était un problème général rencontrer partout dans le monde et normalement c'est pas possible enfin on a tout essayer avec docker (https://stackoverflow.com/questions/26792185/gcloud-auth-login-with-docker-does-not-work-as-it-says-in-documentation)

Config possible

Tu peux lancer avec :

    --https pour HTTPS

    --port pour choisir ton port

    --cert et --key si tu veux ton SSL à toi

Tech utilisées

    Go

    SQLite

    bcrypt + JWT + OAuth2 pour la sécu

    HTML, CSS, JS pour le front

Note

C’est un projet étudiant fait à Ynov Montpellier par Kenzi (Sekkay) et Anas (GLM).
Stack Overflow
gcloud auth login with Docker does not work as it says in documenta...
I've followed the Docker instructions from here exactly: https://cloud.google.com/sdk/#install-docker (click Alternative Methods to find Docker instructions).

But when I run:

docker run -t -i --
