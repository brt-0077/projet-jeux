package main

import (
	"fmt"
	"html/template"
	"math/rand"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type webApp struct {
	mu       sync.Mutex
	state    GameState
	flashMsg string
}

type shopEntry struct {
	Name  string
	Price int
}

type pageData struct {
	State          GameState
	Flash          string
	AttackMode     string
	AttackDamage   int
	FireballDamage int
	MagicReady     bool
	MagicWait      string
	RestartText    string
	CanRestart     bool
	Shop           []shopEntry
}

var pageTpl = template.Must(template.New("page").Parse(`<!doctype html>
<html lang="fr">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width,initial-scale=1">
  <title>RPG Go</title>
  <style>
    :root { --bg:#0d1117; --card:#161b22; --line:#30363d; --text:#e6edf3; --muted:#9da7b3; --accent:#3fb950; --warn:#d29922; }
    body { margin:0; font-family:Segoe UI,Tahoma,sans-serif; background:linear-gradient(150deg,#0b1020,#0d1117); color:var(--text); }
    .wrap { max-width:1100px; margin:24px auto; padding:0 16px 24px; }
    .card { background:var(--card); border:1px solid var(--line); border-radius:12px; padding:14px; margin-bottom:14px; }
    h1,h2,h3,p { margin:0 0 8px; }
    .muted { color:var(--muted); }
    .grid { display:grid; grid-template-columns:repeat(auto-fit,minmax(260px,1fr)); gap:12px; }
    .row { display:flex; flex-wrap:wrap; gap:8px; align-items:center; }
    input,select,button { background:#0d1117; color:var(--text); border:1px solid var(--line); border-radius:8px; padding:8px 10px; }
    button { cursor:pointer; }
    button:hover { border-color:#58a6ff; }
    .pill { display:inline-block; padding:4px 8px; border-radius:999px; border:1px solid var(--line); margin-right:6px; margin-bottom:6px; }
    .log { max-height:220px; overflow:auto; margin:0; padding-left:18px; }
    .flash { border-color:#1f6f3f; background:#122117; color:#9be9a8; }
    .danger { border-color:#8b1f2f; color:#ffb3bd; }
    .shop-item, .inv-item { display:flex; justify-content:space-between; align-items:center; gap:8px; border-top:1px solid var(--line); padding:8px 0; }
    .shop-item:first-child, .inv-item:first-child { border-top:none; }
  </style>
</head>
<body>
  <div class="wrap">
    <div class="card">
      <h1>RPG Go - Interface HTML</h1>
      <p class="muted">Le jeu tourne sur Go. Cette page est servie par le binaire.</p>
      {{if .Flash}}<p class="pill flash">{{.Flash}}</p>{{end}}
      {{if le .State.Player.HP 0}}<p class="pill danger">Vous etes mort</p>{{end}}
    </div>

    <div class="grid">
      <section class="card">
        <h2>Joueur</h2>
        <p><strong>{{.State.Player.Name}}</strong> ({{.State.Player.Class}})</p>
        <p class="muted">Niveau {{.State.Player.Level}} | XP {{.State.Player.Exp}} / {{.State.Player.MaxExp}} | Or {{.State.Player.Gold}}</p>
        <p>HP {{.State.Player.HP}} / {{.State.Player.MaxHP}} | Mana {{.State.Player.Mana}} / {{.State.Player.MaxMana}}</p>
        <p>Arme {{if .State.Player.Equip.Weapon}}{{.State.Player.Equip.Weapon}}{{else}}Coup de poing{{end}} | Ammo {{.State.Player.Ammo}} | Laser {{.State.Player.LaserShots}}</p>

        <form method="post" action="/action" class="row">
          <input type="hidden" name="do" value="rename">
          <input type="text" name="name" maxlength="24" placeholder="Nouveau nom">
          <button type="submit">Renommer</button>
        </form>

        <form method="post" action="/action" class="row" style="margin-top:8px;">
          <input type="hidden" name="do" value="character">
          <select name="id">
            <option value="viking">Ragnar (Viking)</option>
            <option value="militaire">Orion (Soldat)</option>
            <option value="zeusbot">Zephyr (Robot)</option>
          </select>
          <button type="submit">Changer personnage</button>
        </form>
      </section>

      <section class="card">
        <h2>Monstre</h2>
        <p><strong>{{.State.Monster.Name}}</strong></p>
        <p class="muted">Pouvoir: {{.State.Monster.MagicPower}}</p>
        <p>HP {{.State.Monster.HP}} / {{.State.Monster.MaxHP}} | ATK {{.State.Monster.Atk}}</p>
      </section>
    </div>

    <div class="grid">
      <section class="card">
        <h2>Combat</h2>
        <div class="row">
          <form method="post" action="/action"><input type="hidden" name="do" value="attack"><button type="submit">Attaque ({{.AttackMode}} {{.AttackDamage}} dmg)</button></form>
          <form method="post" action="/action"><input type="hidden" name="do" value="fireball"><button type="submit">Boule de feu ({{.FireballDamage}} dmg)</button></form>
          <form method="post" action="/action"><input type="hidden" name="do" value="magic"><button type="submit">Pouvoir magique</button></form>
          <form method="post" action="/action"><input type="hidden" name="do" value="grenade"><button type="submit">Grenade</button></form>
          {{if .CanRestart}}
          <form method="post" action="/action"><input type="hidden" name="do" value="restart"><button type="submit">{{.RestartText}}</button></form>
          {{end}}
        </div>
        {{if not .MagicReady}}<p class="muted" style="margin-top:8px;">Pouvoir magique en recharge: {{.MagicWait}}</p>{{end}}
      </section>

      <section class="card">
        <h2>Actions</h2>
        <div class="row">
          <form method="post" action="/action"><input type="hidden" name="do" value="save"><button type="submit">Sauvegarder</button></form>
          <form method="post" action="/action"><input type="hidden" name="do" value="load"><button type="submit">Charger</button></form>
          <form method="post" action="/action"><input type="hidden" name="do" value="reset"><button type="submit">Reset partie</button></form>
        </div>
      </section>
    </div>

    <div class="grid">
      <section class="card">
        <h2>Inventaire</h2>
        {{if .State.Player.Inventory}}
          {{range .State.Player.Inventory}}
          <div class="inv-item">
            <span>{{.}}</span>
            <form method="post" action="/action">
              <input type="hidden" name="do" value="use_item">
              <input type="hidden" name="item" value="{{.}}">
              <button type="submit">Utiliser</button>
            </form>
          </div>
          {{end}}
        {{else}}
          <p class="muted">Inventaire vide.</p>
        {{end}}
      </section>

      <section class="card">
        <h2>Marchand</h2>
        {{range .Shop}}
        <div class="shop-item">
          <span>{{.Name}} - {{.Price}} or</span>
          <form method="post" action="/action">
            <input type="hidden" name="do" value="buy">
            <input type="hidden" name="item" value="{{.Name}}">
            <button type="submit">Acheter</button>
          </form>
        </div>
        {{end}}
      </section>
    </div>

    <section class="card">
      <h2>Journal</h2>
      <ol class="log">
        {{range .State.CombatLog}}<li>{{.}}</li>{{end}}
      </ol>
    </section>
  </div>
</body>
</html>`))

func runWebServer() {
	rand.Seed(time.Now().UnixNano())
	state, err := loadState(saveFilePath)
	if err != nil {
		state = newState(characterPresets[0].ID)
	}
	app := &webApp{state: state}

	http.HandleFunc("/", app.handleIndex)
	http.HandleFunc("/action", app.handleAction)

	addr := ":8080"
	fmt.Printf("Serveur web lance sur http://localhost%s\n", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		fmt.Printf("Erreur serveur: %v\n", err)
	}
}

func (a *webApp) handleIndex(w http.ResponseWriter, _ *http.Request) {
	a.mu.Lock()
	defer a.mu.Unlock()

	mode, dmg, _ := attackProfile(a.state.Player)
	magicReady := !time.Now().Before(a.state.Player.MagicCooldownUntil)
	magicWait := "0s"
	if !magicReady {
		magicWait = time.Until(a.state.Player.MagicCooldownUntil).Round(time.Second).String()
	}
	canRestart, restartText := webRestartButton(a.state)

	data := pageData{
		State:          a.state,
		Flash:          a.flashMsg,
		AttackMode:     mode,
		AttackDamage:   dmg,
		FireballDamage: fireballDamage(a.state.Player),
		MagicReady:     magicReady,
		MagicWait:      magicWait,
		RestartText:    restartText,
		CanRestart:     canRestart,
		Shop:           shopCatalog(),
	}
	a.flashMsg = ""

	if err := pageTpl.Execute(w, data); err != nil {
		http.Error(w, "Erreur rendu HTML", http.StatusInternalServerError)
	}
}

func (a *webApp) handleAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	_ = r.ParseForm()
	action := r.FormValue("do")
	item := r.FormValue("item")
	name := strings.TrimSpace(r.FormValue("name"))
	id := r.FormValue("id")

	a.mu.Lock()
	switch action {
	case "attack":
		a.flashMsg = a.doAttack()
	case "fireball":
		a.flashMsg = a.doFireball()
	case "magic":
		a.flashMsg = a.doMagic()
	case "grenade":
		a.flashMsg = a.doGrenade()
	case "restart":
		a.flashMsg = a.doRestart()
	case "use_item":
		if item == "" {
			a.flashMsg = "Objet invalide."
		} else {
			a.flashMsg = useItem(&a.state, item)
		}
	case "buy":
		a.flashMsg = a.doBuy(item)
	case "character":
		a.flashMsg = a.doChangeCharacter(id)
	case "rename":
		a.flashMsg = a.doRename(name)
	case "save":
		if err := saveState(saveFilePath, a.state); err != nil {
			a.flashMsg = "Erreur sauvegarde."
		} else {
			a.flashMsg = "Sauvegarde terminee."
		}
	case "load":
		loaded, err := loadState(saveFilePath)
		if err != nil {
			a.flashMsg = "Erreur chargement."
		} else {
			a.state = loaded
			a.flashMsg = "Sauvegarde chargee."
		}
	case "reset":
		a.state = newState(characterPresets[0].ID)
		a.flashMsg = "Partie reinitialisee."
	default:
		a.flashMsg = "Action inconnue."
	}
	normalizeState(&a.state)
	if err := saveState(saveFilePath, a.state); err != nil && action != "load" {
		a.flashMsg = a.flashMsg + " (auto-save KO)"
	}
	a.mu.Unlock()

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (a *webApp) doAttack() string {
	if a.state.Player.HP <= 0 {
		return "Vous etes mort."
	}
	mode, dmg, consume := attackProfile(a.state.Player)
	if consume == "ammo" {
		a.state.Player.Ammo--
	}
	if consume == "laser" {
		a.state.Player.LaserShots--
	}
	a.state.Monster.HP -= dmg
	pushLog(&a.state, fmt.Sprintf("%s inflige %d degats.", mode, dmg))
	triggerPlayerMagic(&a.state, mode)
	if a.state.Monster.HP <= 0 {
		rewardVictory(&a.state)
		return "Victoire."
	}
	enemyTurn(&a.state)
	if a.state.Player.HP <= 0 {
		return "Vous etes mort."
	}
	return "Attaque executee."
}

func (a *webApp) doFireball() string {
	if a.state.Player.HP <= 0 {
		return "Vous etes mort."
	}
	if a.state.Player.Mana < 5 {
		pushLog(&a.state, "Mana insuffisant pour Boule de feu.")
		return "Mana insuffisant."
	}
	dmg := fireballDamage(a.state.Player)
	a.state.Player.Mana -= 5
	a.state.Monster.HP -= dmg
	pushLog(&a.state, fmt.Sprintf("Boule de feu: %d degats.", dmg))
	triggerPlayerMagic(&a.state, "Boule de feu")
	if a.state.Monster.HP <= 0 {
		rewardVictory(&a.state)
		return "Victoire."
	}
	enemyTurn(&a.state)
	if a.state.Player.HP <= 0 {
		return "Vous etes mort."
	}
	return "Boule de feu lancee."
}

func (a *webApp) doMagic() string {
	if a.state.Player.HP <= 0 {
		return "Vous etes mort."
	}
	if time.Now().Before(a.state.Player.MagicCooldownUntil) {
		remaining := time.Until(a.state.Player.MagicCooldownUntil).Round(time.Second)
		return "Pouvoir en recharge: " + remaining.String()
	}
	castPlayerMagicPower(&a.state)
	if a.state.Monster.HP <= 0 {
		rewardVictory(&a.state)
		return "Victoire."
	}
	enemyTurn(&a.state)
	if a.state.Player.HP <= 0 {
		return "Vous etes mort."
	}
	return "Pouvoir magique lance."
}

func (a *webApp) doGrenade() string {
	if a.state.Player.HP <= 0 {
		return "Vous etes mort."
	}
	msg := useItem(&a.state, "Grenade")
	if a.state.Monster.HP > 0 {
		enemyTurn(&a.state)
	}
	return msg
}

func (a *webApp) doBuy(item string) string {
	cost, ok := shopPrices[item]
	if !ok {
		return "Objet inconnu."
	}
	if a.state.Player.Gold < cost {
		return "Pas assez d'or."
	}
	a.state.Player.Gold -= cost
	a.state.Player.Inventory = append(a.state.Player.Inventory, item)
	return "Achat: " + item + " pour " + strconv.Itoa(cost) + " or."
}

func (a *webApp) doChangeCharacter(id string) string {
	if id == "" {
		return "Personnage invalide."
	}
	for _, p := range characterPresets {
		if p.ID == id {
			a.state = newState(id)
			return "Personnage actif: " + p.Name
		}
	}
	return "Personnage introuvable."
}

func (a *webApp) doRename(name string) string {
	if name == "" {
		return "Nom vide ignore."
	}
	if len(name) > 24 {
		name = name[:24]
	}
	a.state.Player.Name = name
	return "Nom mis a jour."
}

func (a *webApp) doRestart() string {
	if a.state.Player.HP > 0 {
		return "Le joueur est deja en vie."
	}
	now := time.Now()
	rc := &a.state.RestartControl
	if !rc.CooldownUntil.IsZero() && now.After(rc.CooldownUntil) {
		rc.Used = 0
		rc.CooldownUntil = time.Time{}
	}
	if !rc.CooldownUntil.IsZero() && now.Before(rc.CooldownUntil) {
		return "Recommencer disponible dans " + time.Until(rc.CooldownUntil).Round(time.Second).String()
	}
	if rc.Used >= restartLimit {
		rc.CooldownUntil = now.Add(restartCooldown)
		return "Limite de tentatives atteinte."
	}
	rc.Used++
	if rc.Used >= restartLimit {
		rc.CooldownUntil = now.Add(restartCooldown)
	}
	cid := a.state.Player.CharacterID
	if cid == "" {
		cid = characterPresets[0].ID
	}
	fresh := playerFromCharacter(cid)
	a.state.Player = fresh
	a.state.Monster = monsterForLevel(fresh.Level)
	a.state.CombatLog = []string{"Nouvelle tentative lancee."}
	return "Recommence avec succes."
}

func webRestartButton(state GameState) (bool, string) {
	if state.Player.HP > 0 {
		return false, ""
	}
	now := time.Now()
	rc := state.RestartControl
	if !rc.CooldownUntil.IsZero() && now.Before(rc.CooldownUntil) {
		return true, "Recommencer dans " + time.Until(rc.CooldownUntil).Round(time.Second).String()
	}
	left := restartLimit - rc.Used
	if left < 0 {
		left = 0
	}
	return true, "Recommencer (" + strconv.Itoa(left) + "/" + strconv.Itoa(restartLimit) + ")"
}

func shopCatalog() []shopEntry {
	items := make([]shopEntry, 0, len(shopPrices))
	for name, price := range shopPrices {
		items = append(items, shopEntry{Name: name, Price: price})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Price == items[j].Price {
			return items[i].Name < items[j].Name
		}
		return items[i].Price < items[j].Price
	})
	return items
}
