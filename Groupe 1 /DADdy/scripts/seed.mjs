#!/usr/bin/env node
// Jeu d'essai pour le front : crée des utilisateurs (avec avatars), des posts
// (avec ou sans images), des commentaires (réponses à un post ou à un autre
// commentaire), des likes et des abonnements — le tout via l'API gateway, afin
// de respecter le hachage des mots de passe, la génération des IDs `usr_`, le
// stockage MinIO et les compteurs gérés par les services.
//
// Usage :
//   node scripts/seed.mjs            # jeu complet (~30 users, ~150 posts, ~400 commentaires)
//   GATEWAY_URL=http://localhost:3001 node scripts/seed.mjs
//   node scripts/seed.mjs --force    # recrée posts/commentaires même si le feed n'est pas vide
//   node scripts/seed.mjs --no-images
//   node scripts/seed.mjs --no-clusters  # engagements aléatoires (sans préférences)
//   node scripts/seed.mjs --users 10 --posts 40 --comments 80
//
// Par défaut, chaque user reçoit 1–2 centres d'intérêt et engage surtout du
// contenu de ces thèmes : des préférences distinctes émergent ainsi dans
// l'algorithme « Pour Toi » (le post-service les matérialise via user_affinity).
//
// Pré-requis : la stack doit tourner (npm run compose -- up -d) et un accès
// internet est nécessaire pour télécharger les images placeholder (pravatar /
// dicebear / picsum). Node >= 18 (fetch / FormData / Blob natifs).

import { writeFile } from "node:fs/promises";
import { resolve } from "node:path";

// ── Configuration ────────────────────────────────────────────────────────────

const GATEWAY_URL = (process.env.GATEWAY_URL ?? "http://localhost:3001").replace(/\/$/, "");
const MAILPIT_URL = (process.env.MAILPIT_URL ?? "http://localhost:8025").replace(/\/$/, "");
const PASSWORD = process.env.SEED_PASSWORD ?? "Password123!";
const TERMS_VERSION = process.env.TERMS_VERSION ?? "2026-06-12";

const argv = process.argv.slice(2);
const hasFlag = (name) => argv.includes(name);
const flagValue = (name, fallback) => {
  const i = argv.indexOf(name);
  return i !== -1 && argv[i + 1] ? Number(argv[i + 1]) : fallback;
};

const FORCE = hasFlag("--force") || process.env.npm_config_force === "true";
const NO_IMAGES = hasFlag("--no-images");
// Par défaut, le seed donne à chaque user des centres d'intérêt (clusters) et
// biaise ses engagements vers ces thèmes, afin que des préférences distinctes
// émergent dans l'algorithme « Pour Toi ». --no-clusters revient à l'ancien
// comportement purement aléatoire (utile pour comparer).
const NO_CLUSTERS = hasFlag("--no-clusters");
const N_USERS = flagValue("--users", 30);
const N_POSTS = flagValue("--posts", 150);
const N_COMMENTS = flagValue("--comments", 400);
const IMAGE_POST_RATIO = 0.4; // ~40 % des posts portent une image
const CONCURRENCY = 6;
const USER_CONCURRENCY = 2; // limité par le rate limiter de /auth/register

// ── Données sources (personas réalistes) ─────────────────────────────────────

const FIRST_NAMES = [
  "Léa", "Hugo", "Camille", "Nathan", "Manon", "Lucas", "Chloé", "Enzo",
  "Sarah", "Théo", "Inès", "Maxime", "Jade", "Antoine", "Louise", "Gabriel",
  "Emma", "Raphaël", "Zoé", "Adam", "Alice", "Noah", "Lina", "Tom",
  "Clara", "Yanis", "Mila", "Sacha", "Eva", "Liam", "Romane", "Ethan",
];
const LAST_NAMES = [
  "Martin", "Bernard", "Dubois", "Robert", "Petit", "Durand", "Leroy", "Moreau",
  "Simon", "Laurent", "Garcia", "David", "Bertrand", "Roux", "Vincent", "Fournier",
  "Morel", "Girard", "André", "Lefèvre", "Mercier", "Blanc", "Guérin", "Boyer",
  "Rousseau", "Chevalier", "Gauthier", "Perrin", "Robin", "Clément", "Morin", "Nicolas",
];
const BIOS = [
  "Développeuse front, fan de café et de rando 🥾",
  "Photographe amateur 📷 | Lyon",
  "J'écris du code et des mauvais jeux de mots.",
  "Passionné de cuisine, de vélo et de séries 📺",
  "Étudiante en design • toujours un carnet à portée de main",
  "Ingénieur le jour, gamer la nuit 🎮",
  "Musicien, lecteur compulsif, amateur de thé 🍵",
  "Je cours après les couchers de soleil 🌅",
  "Chef de projet • adepte du télétravail et des chats",
  "Voyageuse dans l'âme ✈️ | 23 pays et ça continue",
  "Apprenti jardinier urbain 🌱",
  "Sportive, lève-tôt, accro aux podcasts 🎧",
  "Bricoleur du dimanche, perfectionniste en semaine 🔧",
  "Amoureuse des librairies indépendantes 📚",
  "Curieux de tout, expert de rien.",
  "Café > sommeil ☕ | Bordeaux",
  "Designer UI/UX • le détail fait la différence",
  "Trail runner & amateur de bons fromages 🧀",
  "Je code, donc je suis.",
  "Maman débordée et fière de l'être 💛",
];

// ── Générateurs de contenu ───────────────────────────────────────────────────

const rng = (() => {
  let seed = 1337;
  return () => {
    seed = (seed * 1103515245 + 12345) & 0x7fffffff;
    return seed / 0x7fffffff;
  };
})();
const pick = (arr) => arr[Math.floor(rng() * arr.length)];
const pickN = (arr, n) => {
  const copy = [...arr];
  const out = [];
  while (out.length < n && copy.length) out.push(copy.splice(Math.floor(rng() * copy.length), 1)[0]);
  return out;
};
const chance = (p) => rng() < p;

// ── Thèmes & clusters d'intérêt (algorithme « Pour Toi ») ────────────────────
// INTEREST_THEMES doit rester synchronisé avec allowedThemes (post-service) et le
// front (src/lib/themes.ts). On restreint les centres d'intérêt des users aux
// thèmes réellement couverts par le contenu seedé, pour que la personnalisation
// soit visible en démo.
const INTEREST_THEMES = ["tech", "sport", "cuisine", "voyage", "plongee", "lifestyle"];

// Mappe un hashtag (accents retirés, minuscule) → thème.
const HASHTAG_THEME = {
  viededev: "tech", dev: "tech", code: "tech", bug: "tech", debug: "tech",
  git: "tech", prod: "tech", javascript: "tech", ia: "tech", chatgpt: "tech",
  copilot: "tech", linux: "tech", arch: "tech", stackoverflow: "tech",
  framework: "tech", tech: "tech", codereview: "tech",
  sport: "sport", running: "sport", velo: "sport", rando: "sport",
  cuisine: "cuisine", food: "cuisine", ramen: "cuisine", pancakes: "cuisine",
  brunch: "cuisine", apero: "cuisine", batchcooking: "cuisine",
  voyage: "voyage",
  scuba: "plongee", plongee: "plongee",
  cafe: "lifestyle", teletravail: "lifestyle", reunion: "lifestyle",
  jardinage: "lifestyle", productivite: "lifestyle", plantes: "lifestyle",
};

const stripAccents = (s) => s.normalize("NFD").replace(/[\u0300-\u036f]/g, "");

// inferTheme déduit le thème d'un texte de post via son premier hashtag connu.
function inferTheme(text) {
  for (const m of text.matchAll(/#([\p{L}\p{N}_]+)/gu)) {
    const key = stripAccents(m[1].toLowerCase());
    if (HASHTAG_THEME[key]) return HASHTAG_THEME[key];
  }
  return null;
}

// pickAuthorForTheme privilégie (≈80%) un auteur dont les centres d'intérêt
// incluent le thème du post : chaque thème se concentre ainsi chez des auteurs
// alignés, ce qui renforce l'affinité auteur + thème côté lecteurs.
function pickAuthorForTheme(users, theme) {
  if (!NO_CLUSTERS && theme && chance(0.8)) {
    const aligned = users.filter((u) => u.clusters?.includes(theme));
    if (aligned.length) return pick(aligned);
  }
  return pick(users);
}

// pickPostsForUser tire n posts en privilégiant (≈80%) ceux dont le thème fait
// partie des centres d'intérêt du user, le reste au hasard (bruit réaliste).
// C'est le cœur de la génération de préférences : l'engagement clusterisé fait
// émerger des feeds Pour Toi distincts par utilisateur.
function pickPostsForUser(user, postMetas, n) {
  if (NO_CLUSTERS || !user.clusters?.length) {
    return pickN(postMetas, n).map((p) => p.id);
  }
  const inCluster = postMetas.filter((p) => p.theme && user.clusters.includes(p.theme));
  const out = [];
  const seen = new Set();
  while (out.length < n && seen.size < postMetas.length) {
    const pool = inCluster.length && chance(0.8) ? inCluster : postMetas;
    const choice = pool[Math.floor(rng() * pool.length)];
    if (!choice || seen.has(choice.id)) continue;
    seen.add(choice.id);
    out.push(choice.id);
  }
  return out;
}

// pickFollowTargets : abonnements à tendance homophile (≈70% de comptes
// partageant un centre d'intérêt) → nourrit le bonus « abonné » et l'affinité
// auteur du feed Pour Toi. Les abonnements sont posés avant les posts, donc on
// se base sur les clusters des users, pas sur des thèmes de posts.
function pickFollowTargets(user, others, n) {
  if (NO_CLUSTERS || !user.clusters?.length) return pickN(others, n);
  const aligned = others.filter((o) => o.clusters?.some((c) => user.clusters.includes(c)));
  const rest = others.filter((o) => !aligned.includes(o));
  const picked = pickN(aligned, Math.min(aligned.length, Math.round(n * 0.7)));
  return [...picked, ...pickN(rest, Math.max(0, n - picked.length))];
}

// Posts écrits à la main avec hashtags intégrés.
const HANDCRAFTED_POSTS = [
  // dev / tech
  "Je relis du vieux code que j'ai écrit il y a 2 ans. Qui a osé pousser ça en #prod ? Ah. C'était moi. #vieDedev #honte",
  "Le #bug n'était pas dans mon code. Le bug était dans MOI. Un point-virgule. Un seul. #javascript #debug",
  "Déploiement un vendredi à 17h. Je vis dangereusement. #vendredi #prod #yolo #vieDedev",
  "Refactoring du week-end : -400 lignes, mêmes fonctionnalités. Satisfaction maximale 🧹 #code #clean #refactoring",
  "Le PR est ouvert depuis 4 jours. Quelqu'un peut review s'il vous plaît 🙏 #git #codereview #vieDedev",
  "Cette mise à jour a cassé trois trucs pour en réparer un. Classique. #dev #update #bug",
  "Petit rappel à moi-même : commit early, commit often. Et surtout : pas le vendredi soir. #git #vieDedev",
  "Test du nouveau framework JS ce matin : le 47ème framework pour faire un bouton. #javascript #dev #framework",
  "Migration de la base terminée sans casse. On respire 😮‍💨 #postgresql #migration #dev #victoire",
  "Astuce du jour : Cmd+Shift+T pour rouvrir l'onglet que vous venez de fermer par erreur. De rien. #tips #productivité",
  "J'ai installé Arch Linux sur ma machine de travail. Mes collègues ne me reparleront plus jamais. #linux #arch #vieDedev",
  "La réponse à tous mes problèmes était sur #stackoverflow depuis 2009. La honte. #dev #debug",
  "Mon manager m'a demandé si ça allait prendre longtemps. J'ai dit 2 jours. Semaine 3. #vieDedev #estimations #agile",
  "L'IA a généré 200 lignes de code parfaitement inutiles avec une grande confiance. Impressionnant. #ia #code #chatgpt",
  "Dark mode partout sinon rien. Mes yeux ont des droits. #darkmode #dev #santé",
  "Mon code compile du premier coup. Je n'ai pas encore trouvé le bug caché mais il est là. Je le sais. #vieDedev #paranoïa",
  "J'ai passé 3h sur ce bug. C'était une espace insécable. Je démissionne. Non. Mais presque. #debug #vieDedev #unicode",
  "Quelqu'un m'a dit que mon code était pas lisible. Il a raison. Même moi je comprends plus. #code #clean #vieDedev",
  // café / quotidien
  "Le café d'en bas a augmenté ses prix de 20 centimes. C'est la révolution intérieure. #café #inflation #scandale",
  "Trop de café tue le café. Découverte du jour. Tremblement n°4 de la journée. #café #vieDedev #santé",
  "Premier jour de télétravail et déjà 3 réunions avant midi 😅 #télétravail #réunion #vieDedev",
  "J'ai enfin rangé mon bureau. La productivité va exploser. Spoiler : elle n'a pas explosé. #productivité #mensonge",
  "90 % des bugs se règlent avec une bonne nuit de sommeil. Les 10 % restants se règlent avec du café. #debug #café #vieDedev",
  // humour / vie
  "On a adopté un chaton ce week-end. Mon clavier n'a plus peur de rien. Mon chat, si. #chat #chaos #cute",
  "J'ai planté trois basilics sur le rebord de la fenêtre. Un sur deux a déjà abandonné. #jardinage #fail #plantes",
  "Je teste le batch cooking cette semaine. Si ça tient, je deviens officiellement un adulte. #cuisine #batchcooking #adulte",
  "Démo client dans 1h et la feature marche enfin. Je ne touche plus à rien. Rien du tout. #vieDedev #démo #stress",
  "J'écris ce post depuis une réunion qui aurait dû être un email. #réunion #vieDedev #bored",
  "Je viens de liker mon propre post par erreur. Je m'en veux énormément. #malaise #social #oops",
  // sport / perso
  "Nouveau record perso au semi ce matin, les mois d'entraînement paient 🏃 #running #sport #victoire",
  "Première vraie sortie vélo de l'année, les jambes s'en souviendront demain 🚴 #vélo #sport #douleur",
  "Rien de tel qu'une longue rando pour remettre les idées en place 🥾 16 km aujourd'hui ! #rando #nature #détox",
  // sport / entités (hashtags non curatés #messi/#foot/#psg co-engagés avec #sport
  // → ils deviennent « proches » du thème sport via le co-engagement)
  "Quel but de #messi hier soir, quelle légende du #sport ⚽",
  "Le #psg a tout donné, énorme match de #foot ce soir ! #sport",
  "#messi ou #ronaldo, le débat #foot éternel ⚽ #sport",
  "La passe de #messi, un truc pas humain. #foot #sport",
  "Soirée #foot entre potes, le #psg nous a fait vibrer #sport",
  // nourriture
  "Recette de pancakes du dimanche réussie au premier essai. Je suis officiellement un chef 🥞 #cuisine #pancakes #brunch",
  "J'ai testé les ramens du nouveau resto. Mon âme appartient désormais à ce bouillon. #ramen #food #bonheur",
  "Apéro sur la terrasse, le printemps est officiellement lancé 🍷 #apéro #printemps #bonheur",
  // breezy
  "Salut #breezy ! Premier post, ne soyez pas trop méchants 👀 #newbie #social",
  "C'est quoi vos hashtags préférés sur #breezy ? On construit la culture ensemble 🙌 #communauté #hashtags",
  "Le feed #breezy est vraiment bien foutu. Chapeau aux devs 🎩 #breezy #dev #compliment",
  // méta / drôle
  "Quelqu'un peut m'expliquer pourquoi j'ai 0 follower mais 3 suggestions d'abonnement ? #algorithme #lonely #breezy",
  "Post posté à 3h du matin. Productivité nocturne ou mauvaise décision ? Les deux. #nuit #vieDedev #insomnie",
  // scuba
  "Première plongée de l'année ce week-end. 18 mètres, visibilité parfaite, un banc de poissons qui me regarde comme si j'étais le bug. #scuba #plongée #nature",
  "Mon palier de sécurité à 5 mètres : le moment où je règle mentalement tous mes problèmes de code. #scuba #plongée #vieDedev #zen",
  "Quelqu'un a déjà eu un bug en plongée ? Moi oui : mon ordi de plongée a freezé à 20m. J'ai réduit. #scuba #bug #plongée",
  "La plongée c'est le seul endroit où personne peut te demander une réunion. #scuba #plongée #bonheur #télétravail",
  "J'explique à mes collègues que la plongée c'est comme le dev : tu descends, tu explores, et si tu remontes trop vite t'es mort. #scuba #plongée #vieDedev",
];

const TOPICS = {
  thing: ["un bon roman", "une série à binge", "un casque audio", "une appli de productivité", "un podcast tech", "un éditeur de code"],
  place: ["à Lyon", "à Paris", "à Bordeaux", "à Nantes", "à Toulouse", "à Montpellier"],
  food: ["des ramens", "une pizza napolitaine", "un risotto", "des tacos", "du pho", "une raclette"],
  feeling: ["épuisé mais content", "motivé à fond", "plein d'énergie", "complètement à plat", "mystérieusement serein"],
  tag: ["#vieDedev", "#dev", "#télétravail", "#café", "#breezy", "#tech", "#code", "#productivité", "#sport", "#cuisine"],
  emoji: ["😄", "🙌", "🔥", "✨", "☕", "💪", "🤔", "😅", "🌟", "💀"],
};
const TEMPLATES = [
  () => `Quelqu'un aurait une reco pour ${pick(TOPICS.thing)} ? Je sèche complètement ${pick(TOPICS.emoji)} ${pick(TOPICS.tag)}`,
  () => `Journée ${pick(TOPICS.feeling)} aujourd'hui. ${chance(0.5) ? "Et vous ?" : "Hâte d'être à demain."} ${pick(TOPICS.tag)}`,
  () => `J'ai testé ${pick(TOPICS.food)} ${pick(TOPICS.place)} hier soir, franchement validé ${pick(TOPICS.emoji)} #food ${pick(TOPICS.tag)}`,
  () => `Petit point du jour : beaucoup de code, beaucoup de café, zéro résultat visible ${pick(TOPICS.emoji)} #vieDedev #café`,
  () => `On part ${pick(TOPICS.place)} ce week-end, des bons plans ? #voyage ${pick(TOPICS.tag)}`,
  () => `Rien de prévu ce soir à part ${pick(TOPICS.food)} et une série. Le bonheur ${pick(TOPICS.emoji)} #soirée #relax`,
  () => `Note pour plus tard : ne JAMAIS déployer un vendredi ${pick(TOPICS.emoji)} #vendredi #prod #vieDedev`,
  () => `Débat du jour : ${pick(["tabs ou spaces", "café ou thé", "vim ou vscode", "dark mode ou non"])} ? Réponse correcte dans les commentaires. ${pick(TOPICS.tag)}`,
  () => `90 % des bugs disparaissent en dormant dessus. Les 10 % restants deviennent des features. #debug #vieDedev`,
  () => `${pick(["L'IA", "ChatGPT", "Copilot"])} m'a sorti du code parfait. Il ne fait pas ce que je voulais. Parfait. #ia #code #vieDedev`,
  () => `J'ai passé ${pick(["3h", "2h", "une journée entière"])} sur un bug. C'était ${pick(["une virgule en trop", "une majuscule", "un espace", "un import oublié"])}. #vieDedev #debug ${pick(TOPICS.emoji)}`,
  () => `Réunion de 2h qui aurait pu être un email de 3 lignes. ${pick(TOPICS.emoji)} #réunion #vieDedev #télétravail`,
];

// Posts SANS hashtag, regroupés par thème « voulu ». Le post-service ne pourra
// PAS déduire leur thème par hashtag → ils servent à exercer la déduction
// comportementale (inférence d'après les centres d'intérêt des interactants).
// Le thème voulu sert uniquement, côté seed, à orienter qui interagit avec eux.
const THEMED_PLAIN = {
  sport: [
    "Quel match hier soir, intensité de dingue jusqu'à la dernière minute.",
    "Séance de fractionné bouclée ce matin, les jambes parlent encore.",
    "Victoire en prolongation, je n'ai plus de voix mais aucun regret.",
    "Reprise de l'entraînement après deux semaines off, ça pique sérieux.",
  ],
  cuisine: [
    "J'ai enfin réussi ma pâte au levain après trois fournées ratées.",
    "Le secret d'un bon bouillon c'est le temps, et rien d'autre.",
    "Plat mijoté tout l'après-midi, la maison sent divinement bon.",
    "Première tournée de cookies du week-end, déjà à moitié disparue.",
  ],
  plongee: [
    "Visibilité incroyable sur le tombant ce matin, un autre monde là-dessous.",
    "Palier de sécurité respecté, remontée nickel, que du bonheur.",
    "Première sortie en eau froide de la saison, frissons garantis.",
    "Le silence sous l'eau, rien ne vaut ça pour décompresser vraiment.",
  ],
  tech: [
    "Encore un comportement impossible à reproduire en local, évidemment.",
    "Trois heures pour comprendre que c'était un souci de fuseau horaire.",
    "Le refactor du module a tout simplifié, je respire enfin.",
    "La revue de code de ce matin m'a appris deux trucs, franchement top.",
  ],
  voyage: [
    "Réveil face à la baie, le décalage horaire en valait largement la peine.",
    "Train de nuit, carnet ouvert, c'est exactement ça que j'aime.",
    "Perdu dans les ruelles de la vieille ville, le meilleur moyen de visiter.",
    "Dernier jour sur place et déjà l'envie de revenir un jour.",
  ],
  lifestyle: [
    "Premier vrai dimanche sans rien prévu depuis des mois, ça fait du bien.",
    "Grand tri dans l'appart ce week-end, étonnamment libérateur.",
    "Nouvelle plante sur le rebord de la fenêtre, on croise les doigts.",
    "Soirée lecture et tisane, le luxe discret du calme.",
  ],
};

// Posts à hashtag-ENTITÉ non curaté (#messi, #psg…) SANS aucun hashtag de thème :
// le post-service ne peut pas les déduire par dictionnaire ni (souvent) par
// comportement, mais la matrice de proximité reliera #messi à #sport (co-engagé)
// → propagation du thème. theme=null côté seed (engagement non clusterisé).
const ENTITY_POSTS = [
  "#messi est clairement le 🐐, fin du débat.",
  "Personne ne parle assez du dernier match du #psg.",
];

// Construit la liste des posts à créer sous forme {content, theme}. theme est le
// thème « voulu » (issu du hashtag pour les posts taggés, explicite pour les
// posts THEMED_PLAIN) ; il pilote le ciblage des engagements côté seed. Le thème
// réellement stocké est déduit côté serveur.
function buildPostDefs(n) {
  const out = [];
  const seen = new Set();
  const add = (content, theme) => {
    if (content.length > 280 || seen.has(content)) return;
    seen.add(content);
    out.push({ content, theme });
  };
  for (const c of HANDCRAFTED_POSTS) add(c, inferTheme(c));
  for (const [theme, sentences] of Object.entries(THEMED_PLAIN)) {
    for (const s of sentences) add(s, theme);
  }
  // Entités pures : thème voulu null → déduit côté serveur par propagation de voisins.
  for (const c of ENTITY_POSTS) add(c, null);
  let guard = 0;
  while (out.length < n && guard < n * 20) {
    guard++;
    const text = pick(TEMPLATES)();
    add(text, inferTheme(text));
  }
  return out.slice(0, n);
}

const COMMENT_POOL = [
  "Totalement d'accord 👏 #relatable", "Ahah trop vrai 😂", "C'est exactement ma vie #vieDedev",
  "Je connais tellement ça... #vieDedev", "Raconte ! Je suis curieux 👀", "Le mood absolu 😩",
  "Pareil pour moi cette semaine 😅 #solidarity", "Bon courage 💪", "Sérieux ? Faut que j'essaie.",
  "C'est tellement vrai ça fait mal.", "Haha le mood 😄", "Franchement validé 🔥",
  "Le #vendrediProd gang 😅", "On en reparle 👀 #spoiler", "Grave, +1.",
  "Moi j'ai passé 4h sur ce même bug la semaine dernière 😭 #vieDedev",
  "Le pire c'est que je me reconnais à 100 % 💀", "Hâte de voir la suite #suspense",
  "Franchement bien joué 👌", "Ça donne envie 😋", "Je suis pas d'accord mais j'aime le débat 😄",
  "C'est quoi le lien ? #askingForAFriend", "Validé à 100 %.", "Réunion gang 😤 #réunion",
  "Le silence avant le crash en #prod 🔇 #vieDedev", "Compile pas = pas de bug, logique 🤷 #dev",
  "Mon chat a marché sur mon clavier et ça a commité un fichier vide. Meilleur reviewer. #chat #git",
  "J'ai eu exactement le même debug hier. L'espace insécable tue des vies. #dev #unicode",
  "Spoiler : le bug revient la semaine d'après #vieDedev", "T'es sûr que c'était pas du copilot ? 😂 #ia",
  "Le café avant 8h c'est de la survie pas du choix #café #matinal",
];

// ── Client HTTP ──────────────────────────────────────────────────────────────

const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
const MAX_RETRIES = 5; // tentatives max sur 429 avant d'abandonner

let CALLS = 0;
async function api(method, path, { token, json, formData } = {}) {
  CALLS++;
  const headers = { "X-Requested-With": "XMLHttpRequest" };
  if (token) headers.Authorization = `Bearer ${token}`;
  let body;
  if (json !== undefined) {
    headers["Content-Type"] = "application/json";
    body = JSON.stringify(json);
  } else if (formData !== undefined) {
    body = formData; // fetch fixe le Content-Type multipart + boundary
  }

  // Le rate limiter de la gateway peut répondre 429 (notamment sur /auth/* limité
  // par IP). On respecte le header Retry-After et on réessaie, avec un repli
  // exponentiel si le header est absent. FormData n'est pas rejouable (stream
  // consommé) → pas de retry dans ce cas.
  for (let attempt = 0; ; attempt++) {
    const res = await fetch(`${GATEWAY_URL}${path}`, { method, headers, body });
    if (res.status === 429 && formData === undefined && attempt < MAX_RETRIES) {
      const retryAfter = Number(res.headers.get("retry-after"));
      const waitMs = Number.isFinite(retryAfter) && retryAfter > 0 ? retryAfter * 1000 : 2 ** attempt * 500;
      await sleep(waitMs);
      continue;
    }
    const text = await res.text();
    let data;
    try {
      data = text ? JSON.parse(text) : null;
    } catch {
      data = text;
    }
    return { status: res.status, ok: res.ok, data };
  }
}

async function downloadImage(url) {
  const res = await fetch(url, { redirect: "follow" });
  if (!res.ok) throw new Error(`téléchargement image ${url} → HTTP ${res.status}`);
  const contentType = res.headers.get("content-type") ?? "image/jpeg";
  const buffer = Buffer.from(await res.arrayBuffer());
  const ext = contentType.includes("png") ? "png" : contentType.includes("webp") ? "webp" : contentType.includes("gif") ? "gif" : "jpg";
  return { blob: new Blob([buffer], { type: contentType }), ext };
}

// Exécute `worker` sur chaque item avec une concurrence bornée.
async function runPool(items, worker, concurrency = CONCURRENCY) {
  const results = new Array(items.length);
  let cursor = 0;
  const runners = Array.from({ length: Math.min(concurrency, items.length) }, async () => {
    while (cursor < items.length) {
      const i = cursor++;
      results[i] = await worker(items[i], i);
    }
  });
  await Promise.all(runners);
  return results;
}

// ── Phases ───────────────────────────────────────────────────────────────────

function buildUserDefs(n) {
  const used = new Set();
  const users = [];
  for (let i = 0; i < n; i++) {
    const first = FIRST_NAMES[i % FIRST_NAMES.length];
    const last = pick(LAST_NAMES);
    let username = `${first}${last}`.normalize("NFD").replace(/[\u0300-\u036f]/g, "").replace(/[^a-zA-Z0-9]/g, "");
    let base = username;
    let suffix = 1;
    while (used.has(username.toLowerCase())) username = `${base}${++suffix}`;
    used.add(username.toLowerCase());
    // 1 ou 2 centres d'intérêt par user → des profils de préférence distincts.
    const clusters = NO_CLUSTERS ? [] : pickN(INTEREST_THEMES, chance(0.5) ? 1 : 2);
    users.push({
      username,
      email: `${username.toLowerCase()}@example.com`,
      bio: pick(BIOS),
      clusters,
    });
  }
  return users;
}

// Récupère le token de vérification d'email depuis mailpit pour l'adresse donnée.
async function getVerificationToken(email) {
  for (let attempt = 0; attempt < 10; attempt++) {
    await sleep(500);
    const res = await fetch(`${MAILPIT_URL}/api/v1/search?query=to%3A${encodeURIComponent(email)}&limit=1`);
    if (!res.ok) continue;
    const data = await res.json();
    const msg = data.messages?.[0];
    if (!msg) continue;
    const detail = await fetch(`${MAILPIT_URL}/api/v1/message/${msg.ID}`);
    if (!detail.ok) continue;
    const body = await detail.json();
    const match = (body.Text ?? "").match(/token=([A-Za-z0-9_-]+)/);
    if (match) return match[1];
  }
  throw new Error(`token de vérification introuvable dans mailpit pour ${email}`);
}

// Inscrit (ou connecte si déjà présent) un utilisateur, puis pose avatar + bio
// uniquement lorsqu'il vient d'être créé (idempotent au rejeu).
async function ensureUser(def) {
  let created = false;
  let token;
  const reg = await api("POST", "/auth/register", {
    json: { username: def.username, email: def.email, password: PASSWORD, termsAccepted: true, termsVersion: TERMS_VERSION },
  });
  if (reg.status === 201) {
    created = true;
    // Vérifie l'email via mailpit avant de pouvoir se connecter.
    const verifyToken = await getVerificationToken(def.email);
    const verify = await api("POST", "/auth/verify-email", { json: { token: verifyToken } });
    if (!verify.ok) throw new Error(`verify-email ${def.email} échoué (HTTP ${verify.status}): ${JSON.stringify(verify.data)}`);
    const login = await api("POST", "/auth/login", { json: { email: def.email, password: PASSWORD } });
    if (!login.ok) throw new Error(`login post-verify ${def.email} échoué (HTTP ${login.status}): ${JSON.stringify(login.data)}`);
    token = login.data.accessToken;
  } else if (reg.status === 409) {
    let login = await api("POST", "/auth/login", { json: { email: def.email, password: PASSWORD } });
    // Compte créé lors d'un run précédent mais jamais vérifié → on tente de le
    // vérifier via mailpit puis on rejoue le login (idempotence robuste).
    if (login.status === 403) {
      const verifyToken = await getVerificationToken(def.email);
      const verify = await api("POST", "/auth/verify-email", { json: { token: verifyToken } });
      if (!verify.ok) throw new Error(`verify-email ${def.email} échoué (HTTP ${verify.status}): ${JSON.stringify(verify.data)}`);
      login = await api("POST", "/auth/login", { json: { email: def.email, password: PASSWORD } });
    }
    if (!login.ok) throw new Error(`login ${def.email} échoué (HTTP ${login.status}): ${JSON.stringify(login.data)}`);
    token = login.data.accessToken;
  } else {
    throw new Error(`register ${def.email} échoué (HTTP ${reg.status}): ${JSON.stringify(reg.data)}`);
  }

  const me = await api("GET", "/auth/me", { token });
  if (!me.ok) throw new Error(`/auth/me échoué pour ${def.username} (HTTP ${me.status})`);
  const id = me.data.id;

  if (created) {
    await api("PATCH", "/users/me", { token, json: { bio: def.bio } });
    if (!NO_IMAGES) {
      try {
        const { blob, ext } = await downloadImage(`https://i.pravatar.cc/300?u=${encodeURIComponent(def.email)}`);
        const fd = new FormData();
        fd.append("file", blob, `avatar.${ext}`);
        const up = await api("PUT", "/users/me/avatar", { token, formData: fd });
        if (!up.ok) console.warn(`  ⚠ avatar ${def.username} : HTTP ${up.status}`);
      } catch (e) {
        console.warn(`  ⚠ avatar ${def.username} ignoré : ${e.message}`);
      }
    }
  }
  return { ...def, id, token, created };
}

// Crée un post, en téléversant d'abord une image (et en la rattachant via media_ids).
// Le thème n'est jamais envoyé : le post-service le déduit (hashtags, puis
// comportement des interactants).
async function createPost(user, content, withImage) {
  let mediaIds = [];
  if (withImage && !NO_IMAGES) {
    try {
      const seed = Math.floor(rng() * 1e6);
      const { blob, ext } = await downloadImage(`https://picsum.photos/seed/${seed}/800/600`);
      const fd = new FormData();
      fd.append("file", blob, `post.${ext}`);
      const up = await api("POST", "/media/upload", { token: user.token, formData: fd });
      if (up.ok) mediaIds = [up.data.data.id];
      else console.warn(`  ⚠ upload média : HTTP ${up.status}`);
    } catch (e) {
      console.warn(`  ⚠ image de post ignorée : ${e.message}`);
    }
  }
  const res = await api("POST", "/posts", {
    token: user.token,
    json: { content, author_username: user.username, media_ids: mediaIds },
  });
  if (!res.ok) throw new Error(`création post échouée (HTTP ${res.status}): ${JSON.stringify(res.data)}`);
  return res.data.data.id;
}

async function createComment(user, postId, content) {
  const res = await api("POST", `/posts/${postId}/replies`, {
    token: user.token,
    json: { content, author_username: user.username, media_ids: [] },
  });
  if (!res.ok) throw new Error(`commentaire échoué (HTTP ${res.status}): ${JSON.stringify(res.data)}`);
  return res.data.data.id;
}

// ── Main ─────────────────────────────────────────────────────────────────────

async function main() {
  const [maj] = process.versions.node.split(".").map(Number);
  if (maj < 18) {
    console.error(`Node >= 18 requis (fetch/FormData natifs). Version détectée : ${process.versions.node}`);
    process.exit(1);
  }

  console.log(`🌱 Seed Breezy via ${GATEWAY_URL}`);
  console.log(`   users=${N_USERS} posts=${N_POSTS} commentaires=${N_COMMENTS} images=${!NO_IMAGES} force=${FORCE} clusters=${!NO_CLUSTERS}\n`);

  // Vérifie que la gateway répond.
  try {
    await api("GET", "/posts");
  } catch (e) {
    console.error(`❌ Impossible de joindre la gateway (${GATEWAY_URL}). La stack tourne-t-elle ?\n   ${e.message}`);
    process.exit(1);
  }

  // ── 1. Utilisateurs ──
  console.log("👤 Utilisateurs...");
  const userDefs = buildUserDefs(N_USERS);
  const users = await runPool(userDefs, async (def) => {
    try {
      const u = await ensureUser(def);
      process.stdout.write(u.created ? "+" : ".");
      return u;
    } catch (e) {
      console.warn(`\n  ⚠ ${def.username} : ${e.message}`);
      return null;
    }
  }, USER_CONCURRENCY).then((arr) => arr.filter(Boolean));
  console.log(`\n   ${users.length} utilisateurs prêts (${users.filter((u) => u.created).length} créés ce run).`);

  if (users.length === 0) {
    console.error("❌ Aucun utilisateur disponible, arrêt.");
    process.exit(1);
  }

  // Idempotence du contenu : on ne reremplit posts/commentaires que si le feed
  // est vide (sauf --force) pour éviter les doublons au rejeu.
  const feed = await api("GET", "/posts", { token: users[0].token });
  const existing = Array.isArray(feed.data?.data) ? feed.data.data.length : 0;
  if (existing > 0 && !FORCE) {
    console.log(`\nℹ Le feed contient déjà ${existing} posts → posts/commentaires/likes ignorés (relance avec --force pour en rajouter).`);
    await writeCredentials(users);
    summary();
    return;
  }

  // ── 2. Abonnements ──
  console.log("\n🔗 Abonnements...");
  let follows = 0;
  await runPool(users, async (u) => {
    const targets = pickFollowTargets(u, users.filter((o) => o.id !== u.id), 3 + Math.floor(rng() * 6));
    for (const t of targets) {
      const r = await api("POST", `/users/${t.id}/follow`, { token: u.token });
      if (r.ok) follows++;
    }
  });
  console.log(`   ${follows} abonnements créés.`);

  // ── 3. Posts ──
  console.log("\n📝 Posts...");
  const defs = buildPostDefs(N_POSTS);
  const postMetas = await runPool(defs, async (def) => {
    const author = pickAuthorForTheme(users, def.theme);
    try {
      const id = await createPost(author, def.content, chance(IMAGE_POST_RATIO));
      process.stdout.write(".");
      return { id, theme: def.theme };
    } catch (e) {
      console.warn(`\n  ⚠ ${e.message}`);
      return null;
    }
  }).then((arr) => arr.filter(Boolean));
  const postIds = postMetas.map((p) => p.id);
  console.log(`\n   ${postIds.length} posts créés.`);

  // ── 4. Likes ──
  console.log("\n❤️  Likes...");
  let likes = 0;
  await runPool(users, async (u) => {
    for (const pid of pickPostsForUser(u, postMetas, 5 + Math.floor(rng() * 15))) {
      const r = await api("POST", `/posts/${pid}/likes`, { token: u.token });
      if (r.ok) likes++;
    }
  });
  console.log(`   ${likes} likes créés.`);

  // ── 5. Rebreeze ──
  console.log("\n🔁 Rebreeze...");
  let rebreezed = 0;
  await runPool(users, async (u) => {
    for (const pid of pickPostsForUser(u, postMetas, 2 + Math.floor(rng() * 8))) {
      const r = await api("POST", `/posts/${pid}/rebreezers`, { token: u.token });
      if (r.ok || r.status === 201) rebreezed++;
    }
  });
  console.log(`   ${rebreezed} rebreeze créés.`);

  // ── 6. Commentaires (réponses à des posts, parfois à d'autres commentaires) ──
  console.log("\n💬 Commentaires...");
  const tasks = Array.from({ length: N_COMMENTS }, () => {
    const author = pick(users);
    // Commenter en priorité dans ses centres d'intérêt (même biais que les likes).
    const postId = pickPostsForUser(author, postMetas, 1)[0] ?? pick(postIds);
    return { author, postId, content: pick(COMMENT_POOL) };
  });
  const commentIdsByPost = new Map();
  let comments = 0;
  await runPool(tasks, async (t) => {
    // ~30 % des commentaires répondent à un commentaire existant du même post (Fx8).
    let targetId = t.postId;
    const siblings = commentIdsByPost.get(t.postId);
    if (siblings?.length && chance(0.3)) targetId = pick(siblings);
    try {
      const cid = await createComment(t.author, targetId, t.content);
      const arr = commentIdsByPost.get(t.postId) ?? [];
      arr.push(cid);
      commentIdsByPost.set(t.postId, arr);
      comments++;
      process.stdout.write(".");
    } catch (e) {
      console.warn(`\n  ⚠ ${e.message}`);
    }
  }, 4);
  console.log(`\n   ${comments} commentaires créés.`);

  await writeCredentials(users);
  summary();
}

async function writeCredentials(users) {
  const out = resolve("scripts", "seed-users.json");
  await writeFile(
    out,
    JSON.stringify(
      {
        password: PASSWORD,
        gateway: GATEWAY_URL,
        users: users.map((u) => ({ id: u.id, username: u.username, email: u.email, clusters: u.clusters ?? [] })),
      },
      null,
      2,
    ),
  );
  console.log(`\n🔑 Identifiants exportés → ${out}`);
  console.log(`   Tous les comptes partagent le mot de passe : ${PASSWORD}`);
}

function summary() {
  console.log(`\n✅ Terminé (${CALLS} appels API). Le front peut se connecter avec n'importe quel email @example.com.`);
}

main().catch((e) => {
  console.error(`\n❌ Échec : ${e.stack ?? e}`);
  process.exit(1);
});
