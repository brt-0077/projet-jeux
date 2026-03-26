const STATE_KEY = "projet-red-state";
const SAVE_SLOT_KEY = "projet-red-save-slot";

function defaultPlayer() {
  return {
    Name: "Heros",
    Class: "Mage",
    Level: 1,
    HP: 100,
    MaxHP: 100,
    Mana: 10,
    MaxMana: 10,
    Gold: 10,
    Exp: 0,
    MaxExp: 10,
    Turn: 0,
    Inventory: ["Potion", "Potion"],
    Skills: ["Coup de poing"],
    Ammo: 0,
    Equip: {
      Head: "",
      Body: "",
      Feet: "",
      Weapon: "",
    },
  };
}

function maxInt(a, b) {
  return a > b ? a : b;
}

function clampPercent(current, max) {
  if (max <= 0) {
    return 0;
  }
  const percent = Math.floor((current * 100) / max);
  return Math.max(0, Math.min(100, percent));
}

function monsterForLevel(level) {
  const names = ["Gobelin", "Orc", "Troll", "Wyrm", "Demon", "Titan"];
  const name = names[(maxInt(level, 1) - 1) % names.length];
  const maxHP = 70 + level * 22;
  const atk = 7 + level * 3;
  return { Name: name, HP: maxHP, MaxHP: maxHP, Atk: atk };
}

function emptyState() {
  const player = defaultPlayer();
  return {
    player,
    goblin: monsterForLevel(player.Level),
    combatLog: [{ Text: "Le gobelin grogne et se prepare au combat.", Type: "system" }],
    enemyHitFlash: false,
    lastFightHP: player.HP,
    lastFightMana: player.Mana,
    theme: "neonwave",
    soundEnabled: true,
  };
}

function loadState() {
  try {
    const raw = localStorage.getItem(STATE_KEY);
    if (!raw) {
      return emptyState();
    }
    const parsed = JSON.parse(raw);
    const base = emptyState();

    const state = {
      ...base,
      ...parsed,
      player: {
        ...base.player,
        ...(parsed.player || {}),
        Equip: {
          ...base.player.Equip,
          ...((parsed.player && parsed.player.Equip) || {}),
        },
      },
      goblin: {
        ...base.goblin,
        ...(parsed.goblin || {}),
      },
      combatLog: Array.isArray(parsed.combatLog) ? parsed.combatLog : base.combatLog,
    };

    if (!Array.isArray(state.player.Inventory)) state.player.Inventory = [];
    if (!Array.isArray(state.player.Skills)) state.player.Skills = [];
    if (!Number.isFinite(state.player.Ammo)) state.player.Ammo = 0;
    if (!Number.isFinite(state.player.Turn)) state.player.Turn = 0;
    if (typeof state.soundEnabled !== "boolean") state.soundEnabled = true;

    return state;
  } catch (_error) {
    return emptyState();
  }
}

let state = loadState();
let toastAudioContext = null;

function saveState() {
  localStorage.setItem(STATE_KEY, JSON.stringify(state));
}

function playToastSound(type) {
  if (!state.soundEnabled) return;

  const AudioContextClass = window.AudioContext || window.webkitAudioContext;
  if (!AudioContextClass) return;

  if (!toastAudioContext) {
    toastAudioContext = new AudioContextClass();
  }

  if (toastAudioContext.state === "suspended") {
    toastAudioContext.resume();
  }

  const now = toastAudioContext.currentTime;
  const oscillator = toastAudioContext.createOscillator();
  const gain = toastAudioContext.createGain();

  let frequency = 620;
  if (type === "success") frequency = 760;
  if (type === "error") frequency = 280;

  oscillator.type = "triangle";
  oscillator.frequency.setValueAtTime(frequency, now);

  gain.gain.setValueAtTime(0.0001, now);
  gain.gain.exponentialRampToValueAtTime(0.04, now + 0.01);
  gain.gain.exponentialRampToValueAtTime(0.0001, now + 0.12);

  oscillator.connect(gain);
  gain.connect(toastAudioContext.destination);
  oscillator.start(now);
  oscillator.stop(now + 0.12);
}

function getToastStack() {
  let stack = document.getElementById("toast-stack");
  if (stack) return stack;

  stack = document.createElement("div");
  stack.id = "toast-stack";
  stack.className = "toast-stack";
  document.body.appendChild(stack);
  return stack;
}

function showToast(message, type = "info") {
  playToastSound(type);
  const stack = getToastStack();
  const toast = document.createElement("div");
  toast.className = `toast toast-${type}`;
  toast.textContent = message;
  stack.appendChild(toast);

  setTimeout(() => {
    toast.remove();
  }, 2400);
}

function sanitizeTheme(raw) {
  return raw === "neonwave" || raw === "avifwave" || raw === "polygonwave" ? raw : "neonwave";
}

function applyTheme() {
  document.body.classList.remove("theme-neonwave", "theme-avifwave", "theme-polygonwave");
  document.body.classList.add(`theme-${state.theme}`);
}

function resolveThemeFromQuery() {
  const params = new URLSearchParams(window.location.search);
  const queryTheme = params.get("theme");
  if (queryTheme) {
    state.theme = sanitizeTheme(queryTheme);
    saveState();
  }
  state.theme = sanitizeTheme(state.theme);
  applyTheme();
}

function pushCombatMessage(text, type) {
  state.combatLog.push({ Text: text, Type: type });
  if (state.combatLog.length > 8) {
    state.combatLog = state.combatLog.slice(state.combatLog.length - 8);
  }
}

function gainXP(amount) {
  const player = state.player;
  player.Exp += amount;
  if (player.Exp >= player.MaxExp) {
    player.Level += 1;
    player.Exp -= player.MaxExp;
    player.MaxExp += 10;
    player.MaxHP += 5;
    player.MaxMana += 2;
    player.HP = player.MaxHP;
    player.Mana = player.MaxMana;
    state.goblin = monsterForLevel(player.Level);
    pushCombatMessage(`Niveau ${player.Level} atteint. Un ${state.goblin.Name} plus puissant apparait!`, "system");
  }
}

function basicAttackDamage() {
  return maxInt(6, Math.floor(state.player.MaxHP / 10));
}

function fireballDamage() {
  return maxInt(12, Math.floor(state.player.MaxHP / 5));
}

function enemyAttackDamage() {
  return maxInt(state.goblin.Atk, Math.floor(state.player.MaxHP / 12));
}

function attackProfile() {
  const base = basicAttackDamage();
  switch (state.player.Equip.Weapon) {
    case "Pistolet":
      if (state.player.Ammo > 0) {
        return { mode: "Pistolet", damage: maxInt(base + 10, Math.floor(state.player.MaxHP / 4)), usesAmmo: true };
      }
      return { mode: "Pistolet (sans munitions)", damage: base, usesAmmo: false };
    case "Laser":
      return { mode: "Laser", damage: maxInt(base + 18, Math.floor(state.player.MaxHP / 3)), usesAmmo: false };
    default:
      return { mode: "Coup de poing", damage: base, usesAmmo: false };
  }
}

function equippedWeaponName() {
  return state.player.Equip.Weapon || "Coup de poing";
}

function statDeltaState(current, previous) {
  if (current > previous) return "gain";
  if (current < previous) return "loss";
  return "";
}

function iconForItem(item) {
  switch (item) {
    case "Potion": return "🧪";
    case "Poison": return "☠️";
    case "Livre": return "📘";
    case "Casque": return "🪖";
    case "Armure": return "🛡️";
    case "Bottes": return "🥾";
    case "Pistolet": return "🔫";
    case "Laser": return "🟣";
    case "Munitions": return "📦";
    default: return "🎒";
  }
}

function itemDescription(item) {
  switch (item) {
    case "Potion": return "+10 PV (sans depasser le maximum).";
    case "Poison": return "-5 PV sur vous. A utiliser avec prudence.";
    case "Livre": return "Debloque la competence Boule de feu si absente.";
    case "Casque": return "Equipement tete: +5 PV max.";
    case "Armure": return "Equipement corps: +10 PV max.";
    case "Bottes": return "Equipement pieds: +5 PV max.";
    case "Pistolet": return "Arme a feu: gros degats, consomme 1 munition par tir.";
    case "Laser": return "Arme energie: tres gros degats, sans munition.";
    case "Munitions": return "Recharge de 6 munitions pour le pistolet.";
    default: return "Objet de voyage utile selon la situation.";
  }
}

function equipItem(item) {
  switch (item) {
    case "Casque":
      state.player.Equip.Head = item;
      state.player.MaxHP += 5;
      break;
    case "Armure":
      state.player.Equip.Body = item;
      state.player.MaxHP += 10;
      break;
    case "Bottes":
      state.player.Equip.Feet = item;
      state.player.MaxHP += 5;
      break;
    default:
      break;
  }
}

function consumeItemOnce(item) {
  const index = state.player.Inventory.indexOf(item);
  if (index === -1) return false;
  state.player.Inventory.splice(index, 1);
  return true;
}

function useItem(item) {
  switch (item) {
    case "Potion":
      if (!consumeItemOnce("Potion")) return { ok: false, message: "Objet introuvable dans l'inventaire.", type: "error" };
      state.player.HP = Math.min(state.player.MaxHP, state.player.HP + 10);
      return { ok: true, message: "Potion utilisee: +10 PV.", type: "success" };
    case "Poison":
      if (!consumeItemOnce("Poison")) return { ok: false, message: "Objet introuvable dans l'inventaire.", type: "error" };
      state.player.HP -= 5;
      return { ok: true, message: "Poison utilise: -5 PV.", type: "info" };
    case "Livre":
      if (!consumeItemOnce("Livre")) return { ok: false, message: "Objet introuvable dans l'inventaire.", type: "error" };
      if (!state.player.Skills.includes("Boule de feu")) {
        state.player.Skills.push("Boule de feu");
        return { ok: true, message: "Nouvelle competence debloquee: Boule de feu.", type: "success" };
      }
      return { ok: true, message: "Livre utilise: aucune nouvelle competence.", type: "info" };
    case "Casque":
    case "Armure":
    case "Bottes":
      equipItem(item);
      return { ok: true, message: `${item} equipe.`, type: "success" };
    case "Pistolet":
    case "Laser":
      state.player.Equip.Weapon = item;
      return { ok: true, message: `${item} equipe en arme.`, type: "success" };
    case "Munitions":
      if (!consumeItemOnce("Munitions")) return { ok: false, message: "Objet introuvable dans l'inventaire.", type: "error" };
      state.player.Ammo += 6;
      return { ok: true, message: "Munitions chargees: +6.", type: "success" };
    default:
      return { ok: false, message: "Objet non reconnu.", type: "error" };
  }
}

function buyItem(item) {
  const prices = {
    Potion: 3,
    Poison: 6,
    Livre: 25,
    Casque: 10,
    Armure: 20,
    Bottes: 15,
    Pistolet: 40,
    Laser: 65,
    Munitions: 8,
  };
  const cost = prices[item];
  if (!cost) return { ok: false, message: "Objet non reconnu.", type: "error" };
  if (state.player.Gold < cost) {
    return { ok: false, message: `Pas assez d'or pour ${item}.`, type: "error" };
  }
  state.player.Gold -= cost;
  state.player.Inventory.push(item);
  return { ok: true, message: `${item} achete pour ${cost} or.`, type: "success" };
}

function enemyTurn() {
  state.player.Turn += 1;
  let damage = enemyAttackDamage();
  const turnLabel = `Tour ${state.player.Turn}`;

  if (state.player.Turn % 3 === 0) {
    damage *= 2;
    pushCombatMessage(`${turnLabel}: le gobelin enrage et inflige un coup puissant!`, "critical");
  }

  if (state.goblin.HP > 0) {
    state.player.HP -= damage;
    pushCombatMessage(`${turnLabel}: le gobelin attaque et inflige ${damage} degats.`, "enemy");
  }

  if (state.goblin.HP <= 0) {
    state.goblin = monsterForLevel(state.player.Level);
    state.player.Gold += 10;
    gainXP(5);
    state.player.Turn = 0;
    pushCombatMessage(`Victoire! +10 or et +5 XP. Un ${state.goblin.Name} apparait.`, "system");
  }

  if (state.player.HP <= 0) {
    state.player.HP = Math.floor(state.player.MaxHP / 2);
    state.player.Turn = 0;
    pushCombatMessage("Vous tombez au combat... Vous revenez avec la moitie de vos PV.", "warning");
  }
}

function fillHud() {
  const map = {
    "hud-name": state.player.Name,
    "hud-hp": `${state.player.HP} / ${state.player.MaxHP}`,
    "hud-mana": `${state.player.Mana} / ${state.player.MaxMana}`,
    "hud-gold": `${state.player.Gold}`,
  };

  Object.entries(map).forEach(([id, value]) => {
    const el = document.getElementById(id);
    if (el) el.textContent = value;
  });
}

function renderHome() {
  const xpPercent = clampPercent(state.player.Exp, state.player.MaxExp);
  const setText = (id, value) => {
    const el = document.getElementById(id);
    if (el) el.textContent = value;
  };

  setText("hero-name", state.player.Name);
  setText("hero-class", state.player.Class);
  setText("meta-level", state.player.Level);
  setText("meta-xp", `${state.player.Exp} / ${state.player.MaxExp}`);
  setText("meta-skills", state.player.Skills.length);
  setText("meta-items", state.player.Inventory.length);
  setText("meta-weapon", equippedWeaponName());
  setText("meta-ammo", state.player.Ammo);

  const xpBar = document.getElementById("xp-bar");
  if (xpBar) xpBar.style.setProperty("--value", `${xpPercent}%`);

  const nameInput = document.getElementById("name");
  if (nameInput) nameInput.value = state.player.Name;

  const form = document.getElementById("name-form");
  if (form) {
    form.onsubmit = (event) => {
      event.preventDefault();
      const raw = (nameInput.value || "").trim();
      if (raw) {
        state.player.Name = raw.slice(0, 24);
        showToast("Nom du personnage mis a jour.", "success");
      }
      saveState();
      renderHome();
      fillHud();
    };
  }

  const saveLink = document.getElementById("save-link");
  if (saveLink) {
    saveLink.onclick = (event) => {
      event.preventDefault();
      localStorage.setItem(SAVE_SLOT_KEY, JSON.stringify(state.player));
      showToast("Partie sauvegardee.", "success");
    };
  }

  const soundToggle = document.getElementById("sound-toggle");
  if (soundToggle) {
    soundToggle.textContent = `Son: ${state.soundEnabled ? "ON" : "OFF"}`;
    soundToggle.onclick = (event) => {
      event.preventDefault();
      state.soundEnabled = !state.soundEnabled;
      saveState();
      renderHome();
      showToast(
        state.soundEnabled ? "Sons des notifications actives." : "Sons des notifications desactives.",
        "info"
      );
    };
  }

  const loadLink = document.getElementById("load-link");
  if (loadLink) {
    loadLink.onclick = (event) => {
      event.preventDefault();
      const raw = localStorage.getItem(SAVE_SLOT_KEY);
      if (!raw) {
        showToast("Aucune sauvegarde trouvee.", "error");
        return;
      }
      try {
        const player = JSON.parse(raw);
        state.player = {
          ...defaultPlayer(),
          ...player,
          Equip: {
            ...defaultPlayer().Equip,
            ...((player && player.Equip) || {}),
          },
        };
        state.goblin = monsterForLevel(state.player.Level);
        state.lastFightHP = state.player.HP;
        state.lastFightMana = state.player.Mana;
        saveState();
        renderHome();
        fillHud();
        showToast("Sauvegarde chargee.", "success");
      } catch (_error) {
        showToast("Sauvegarde invalide.", "error");
      }
    };
  }

  const resetLink = document.getElementById("reset-link");
  if (resetLink) {
    resetLink.onclick = (event) => {
      event.preventDefault();
      const prevSoundEnabled = state.soundEnabled;
      state = emptyState();
      state.soundEnabled = prevSoundEnabled;
      saveState();
      fillHud();
      renderHome();
      showToast("Partie reinitialisee.", "info");
    };
  }
}

function rarityClass(itemName) {
  if (["Potion", "Poison", "Munitions"].includes(itemName)) return "rarity-common";
  if (["Casque", "Bottes", "Pistolet"].includes(itemName)) return "rarity-rare";
  return "rarity-epic";
}

function rarityTag(itemName) {
  if (["Potion", "Poison", "Munitions"].includes(itemName)) {
    return '<span class="rarity-tag common">Commun</span>';
  }
  if (["Casque", "Bottes", "Pistolet"].includes(itemName)) {
    return '<span class="rarity-tag rare">Rare</span>';
  }
  return '<span class="rarity-tag epic">Epique</span>';
}

function renderInventory() {
  const grid = document.getElementById("inventory-grid");
  if (!grid) return;

  if (state.player.Inventory.length === 0) {
    grid.innerHTML = '<article class="item-card rarity-common"><div class="item-info"><h2>Inventaire vide</h2><p class="item-description">Passez au marchand pour acheter des objets.</p></div></article>';
    return;
  }

  grid.innerHTML = state.player.Inventory
    .map((item) => {
      return `
        <article class="item-card ${rarityClass(item)}">
          <div class="item-icon">${iconForItem(item)}</div>
          <div class="item-info">
            ${rarityTag(item)}
            <h2>${item}</h2>
            <p class="item-description">${itemDescription(item)}</p>
            <button class="btn-primary item-use-btn" data-item="${item}">Utiliser</button>
          </div>
        </article>
      `;
    })
    .join("");

  grid.querySelectorAll(".item-use-btn").forEach((button) => {
    button.addEventListener("click", () => {
      const item = button.getAttribute("data-item");
      const result = useItem(item);
      saveState();
      fillHud();
      renderInventory();
      if (result && result.message) showToast(result.message, result.type);
    });
  });
}

function renderShop() {
  const goldEl = document.getElementById("shop-gold");
  if (goldEl) goldEl.textContent = state.player.Gold;

  document.querySelectorAll(".buy-btn").forEach((button) => {
    button.onclick = () => {
      const item = button.getAttribute("data-item");
      const result = buyItem(item);
      saveState();
      fillHud();
      renderShop();
      if (result && result.message) showToast(result.message, result.type);
    };
  });
}

function setBarState(bar, stateName) {
  if (!bar) return;
  bar.classList.remove("bar-gain", "bar-loss");
  if (stateName === "gain") bar.classList.add("bar-gain");
  if (stateName === "loss") bar.classList.add("bar-loss");
}

function renderFight() {
  const { mode, damage } = attackProfile();
  const hpPercent = clampPercent(state.player.HP, state.player.MaxHP);
  const manaPercent = clampPercent(state.player.Mana, state.player.MaxMana);
  const xpPercent = clampPercent(state.player.Exp, state.player.MaxExp);
  const monsterPercent = clampPercent(state.goblin.HP, state.goblin.MaxHP);

  const hpState = statDeltaState(state.player.HP, state.lastFightHP);
  const manaState = statDeltaState(state.player.Mana, state.lastFightMana);
  state.lastFightHP = state.player.HP;
  state.lastFightMana = state.player.Mana;

  const setText = (id, value) => {
    const el = document.getElementById(id);
    if (el) el.textContent = value;
  };

  setText("player-name-fight", state.player.Name);
  setText("monster-name-fight", state.goblin.Name);
  setText("player-name-card", state.player.Name);
  setText("weapon-name", equippedWeaponName());
  setText("player-ammo", state.player.Ammo);
  setText("player-hp-text", `${state.player.HP} / ${state.player.MaxHP}`);
  setText("player-mana-text", `${state.player.Mana} / ${state.player.MaxMana}`);
  setText("player-xp-text", `${state.player.Exp} / ${state.player.MaxExp}`);
  setText("monster-name-card", state.goblin.Name);
  setText("monster-hp-text", `${state.goblin.HP} / ${state.goblin.MaxHP}`);

  const playerHPBar = document.getElementById("player-hp-bar");
  const playerManaBar = document.getElementById("player-mana-bar");
  const playerXPBar = document.getElementById("player-xp-bar");
  const monsterHPBar = document.getElementById("monster-hp-bar");

  if (playerHPBar) playerHPBar.style.setProperty("--value", `${hpPercent}%`);
  if (playerManaBar) playerManaBar.style.setProperty("--value", `${manaPercent}%`);
  if (playerXPBar) playerXPBar.style.setProperty("--value", `${xpPercent}%`);
  if (monsterHPBar) monsterHPBar.style.setProperty("--value", `${monsterPercent}%`);

  setBarState(playerHPBar, hpState);
  setBarState(playerManaBar, manaState);

  const attackBtn = document.getElementById("attack-btn");
  if (attackBtn) attackBtn.textContent = `${mode} (${damage} dmg)`;

  const fireballBtn = document.getElementById("fireball-btn");
  if (fireballBtn) fireballBtn.textContent = `🔥 Boule de feu (${fireballDamage()} dmg / 5 mana)`;

  const logEl = document.getElementById("combat-log");
  if (logEl) {
    logEl.innerHTML = state.combatLog
      .map((message) => `<li class="log-${message.Type}">${message.Text}</li>`)
      .join("");
  }

  const enemyCard = document.getElementById("enemy-card");
  const enemyFighter = document.getElementById("enemy-fighter");
  if (state.enemyHitFlash) {
    if (enemyCard) enemyCard.classList.add("enemy-hit");
    if (enemyFighter) enemyFighter.classList.add("enemy-hit");
    setTimeout(() => {
      if (enemyCard) enemyCard.classList.remove("enemy-hit");
      if (enemyFighter) enemyFighter.classList.remove("enemy-hit");
    }, 350);
    state.enemyHitFlash = false;
  }

  if (attackBtn) {
    attackBtn.onclick = () => {
      const profile = attackProfile();
      if (profile.usesAmmo) state.player.Ammo -= 1;
      state.goblin.HP -= profile.damage;
      pushCombatMessage(`${profile.mode}: -${profile.damage} PV au gobelin.`, "player");
      state.enemyHitFlash = true;
      enemyTurn();
      saveState();
      fillHud();
      renderFight();
    };
  }

  if (fireballBtn) {
    fireballBtn.onclick = () => {
      if (state.player.Mana >= 5) {
        const damageValue = fireballDamage();
        state.player.Mana -= 5;
        state.goblin.HP -= damageValue;
        pushCombatMessage(`Boule de feu critique: -${damageValue} PV au gobelin.`, "player");
        state.enemyHitFlash = true;
      } else {
        pushCombatMessage("Mana insuffisant pour lancer Boule de feu.", "warning");
      }
      enemyTurn();
      saveState();
      fillHud();
      renderFight();
    };
  }
}

function detectPage() {
  const marker = document.body.getAttribute("data-page");
  if (marker) return marker;

  const fileName = window.location.pathname.split("/").pop().toLowerCase();
  if (fileName === "inventory.html") return "inventory";
  if (fileName === "shop.html") return "shop";
  if (fileName === "fight.html") return "fight";
  return "home";
}

function bootstrap() {
  resolveThemeFromQuery();
  fillHud();

  const page = detectPage();
  if (page === "home") renderHome();
  if (page === "inventory") renderInventory();
  if (page === "shop") renderShop();
  if (page === "fight") renderFight();

  saveState();
}

document.addEventListener("DOMContentLoaded", bootstrap);
