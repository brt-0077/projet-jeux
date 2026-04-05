package main

import (
	"bufio"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"math"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	saveFilePath         = "save.gob"
	restartLimit         = 3
	restartCooldown      = 10 * time.Minute
	magicCooldown        = 15 * time.Second
	grenadeDamage        = 22
	consumablePotionHeal = 10
)

type CharacterPreset struct {
	ID         string
	Name       string
	Class      string
	MagicPower string
	MaxHP      int
	MaxMana    int
	Gold       int
	Skills     []string
}

type Equipment struct {
	Head   string `json:"Head"`
	Body   string `json:"Body"`
	Feet   string `json:"Feet"`
	Weapon string `json:"Weapon"`
}

type Player struct {
	CharacterID        string    `json:"CharacterId"`
	Name               string    `json:"Name"`
	Class              string    `json:"Class"`
	MagicPower         string    `json:"MagicPower"`
	Level              int       `json:"Level"`
	HP                 int       `json:"HP"`
	MaxHP              int       `json:"MaxHP"`
	Mana               int       `json:"Mana"`
	MaxMana            int       `json:"MaxMana"`
	Gold               int       `json:"Gold"`
	Exp                int       `json:"Exp"`
	MaxExp             int       `json:"MaxExp"`
	Turn               int       `json:"Turn"`
	Inventory          []string  `json:"Inventory"`
	Skills             []string  `json:"Skills"`
	MagicCooldownUntil time.Time `json:"MagicCooldownUntil"`
	Ammo               int       `json:"Ammo"`
	LaserShots         int       `json:"LaserShots"`
	Equip              Equipment `json:"Equip"`
}

type Monster struct {
	Name       string
	HP         int
	MaxHP      int
	Atk        int
	ArenaTier  int
	MagicPower string
}

type RestartControl struct {
	Used          int       `json:"used"`
	CooldownUntil time.Time `json:"cooldownUntil"`
}

type GameState struct {
	Player         Player         `json:"player"`
	Monster        Monster        `json:"monster"`
	CombatLog      []string       `json:"combatLog"`
	RestartControl RestartControl `json:"restartControl"`
	SoundEnabled   bool           `json:"soundEnabled"`
	Theme          string         `json:"theme"`
}

var characterPresets = []CharacterPreset{
	{
		ID:         "viking",
		Name:       "Ragnar",
		Class:      "Viking",
		MagicPower: "Rage runique",
		MaxHP:      115,
		MaxMana:    8,
		Gold:       50,
		Skills:     []string{"Coup de poing"},
	},
	{
		ID:         "militaire",
		Name:       "Orion",
		Class:      "Soldat",
		MagicPower: "Aura tactique",
		MaxHP:      100,
		MaxMana:    10,
		Gold:       50,
		Skills:     []string{"Coup de poing"},
	},
	{
		ID:         "zeusbot",
		Name:       "Zephyr",
		Class:      "Robot",
		MagicPower: "Surcharge arcane",
		MaxHP:      92,
		MaxMana:    16,
		Gold:       50,
		Skills:     []string{"Coup de poing", "Boule de feu"},
	},
}

var shopPrices = map[string]int{
	"Potion":          3,
	"Poison":          6,
	"Livre":           25,
	"Casque":          10,
	"Armure":          20,
	"Bottes":          15,
	"Pistolet":        40,
	"Epee":            22,
	"Laser":           65,
	"Lance-roquettes": 120,
	"Munitions":       8,
	"Batterie":        12,
	"Grenade":         18,
}

func main() {
	runCLI()
}

func runCLI() {
	rand.Seed(time.Now().UnixNano())

	state, err := loadState(saveFilePath)
	if err != nil {
		fmt.Printf("Impossible de charger la sauvegarde (%v). Nouvelle partie.\n", err)
		state = newState(characterPresets[0].ID)
	}

	scanner := bufio.NewScanner(os.Stdin)
	for {
		if state.Player.HP <= 0 {
			handleDeath(&state, scanner)
		}

		printStatus(state)
		fmt.Println("\nMenu:")
		fmt.Println("1) Combat")
		fmt.Println("2) Inventaire")
		fmt.Println("3) Marchand")
		fmt.Println("4) Changer de personnage")
		fmt.Println("5) Changer nom")
		fmt.Println("6) Sauvegarder")
		fmt.Println("7) Charger")
		fmt.Println("8) Quitter")
		fmt.Print("Choix > ")

		choice := readChoice(scanner)
		switch choice {
		case "1":
			fightLoop(&state, scanner)
		case "2":
			inventoryLoop(&state, scanner)
		case "3":
			shopLoop(&state, scanner)
		case "4":
			chooseCharacter(&state, scanner)
		case "5":
			changeName(&state, scanner)
		case "6":
			if err := saveState(saveFilePath, state); err != nil {
				fmt.Printf("Erreur sauvegarde: %v\n", err)
			} else {
				fmt.Println("Sauvegarde terminee.")
			}
		case "7":
			loaded, err := loadState(saveFilePath)
			if err != nil {
				fmt.Printf("Erreur chargement: %v\n", err)
			} else {
				state = loaded
				fmt.Println("Sauvegarde chargee.")
			}
		case "8":
			if err := saveState(saveFilePath, state); err != nil {
				fmt.Printf("Sauvegarde finale echouee: %v\n", err)
			}
			fmt.Println("A bientot.")
			return
		default:
			fmt.Println("Choix invalide.")
		}
	}
}

func newState(characterID string) GameState {
	player := playerFromCharacter(characterID)
	monster := monsterForLevel(player.Level)
	return GameState{
		Player:       player,
		Monster:      monster,
		CombatLog:    []string{fmt.Sprintf("Un %s apparait.", monster.Name)},
		SoundEnabled: true,
		Theme:        "neonwave",
	}
}

func playerFromCharacter(id string) Player {
	preset := characterPresets[0]
	for _, p := range characterPresets {
		if p.ID == id {
			preset = p
			break
		}
	}

	inv := []string{"Potion", "Potion"}
	skills := make([]string, len(preset.Skills))
	copy(skills, preset.Skills)
	return Player{
		CharacterID: preset.ID,
		Name:        preset.Name,
		Class:       preset.Class,
		MagicPower:  preset.MagicPower,
		Level:       1,
		HP:          preset.MaxHP,
		MaxHP:       preset.MaxHP,
		Mana:        preset.MaxMana,
		MaxMana:     preset.MaxMana,
		Gold:        preset.Gold,
		Exp:         0,
		MaxExp:      10,
		Inventory:   inv,
		Skills:      skills,
		Equip:       Equipment{},
	}
}

func monsterForLevel(level int) Monster {
	if level < 1 {
		level = 1
	}
	names := []string{"Gobelin", "Orc", "Troll", "Wyrm", "Demon", "Titan"}
	arena := level
	if arena > len(names) {
		arena = len(names)
	}
	name := names[arena-1]
	maxHP := 60 + level*20 + arena*16
	atk := 6 + level*3 + arena*2
	return Monster{
		Name:       name,
		HP:         maxHP,
		MaxHP:      maxHP,
		Atk:        atk,
		ArenaTier:  arena,
		MagicPower: monsterMagic(name),
	}
}

func monsterMagic(name string) string {
	switch name {
	case "Gobelin":
		return "Crachat toxique"
	case "Orc":
		return "Cri de guerre"
	case "Troll":
		return "Regeneration brute"
	case "Wyrm":
		return "Souffle draconique"
	case "Demon":
		return "Flamme infernale"
	case "Titan":
		return "Onde cataclysmique"
	default:
		return "Furie occulte"
	}
}

func basicAttackDamage(p Player) int {
	return maxInt(6, p.MaxHP/10)
}

func fireballDamage(p Player) int {
	return maxInt(12, p.MaxHP/5)
}

func enemyAttackDamage(p Player, m Monster) int {
	return maxInt(m.Atk, p.MaxHP/12)
}

func attackProfile(p Player) (mode string, damage int, consume string) {
	base := basicAttackDamage(p)
	switch p.Equip.Weapon {
	case "Epee":
		return "Epee", maxInt(base+4, p.MaxHP/6), "none"
	case "Pistolet":
		if p.Ammo > 0 {
			return "Pistolet", maxInt(base+10, p.MaxHP/4), "ammo"
		}
		return "Pistolet (sans munitions)", base, "none"
	case "Laser":
		if p.LaserShots > 0 {
			return "Laser", maxInt(base+18, p.MaxHP/3), "laser"
		}
		return "Laser (sans batterie)", base, "none"
	case "Lance-roquettes":
		if p.Ammo > 0 {
			return "Lance-roquettes", maxInt(base+16, int(math.Floor(float64(p.MaxHP)/2.5))), "ammo"
		}
		return "Lance-roquettes (sans munitions)", base, "none"
	default:
		return "Coup de poing", base, "none"
	}
}

func fightLoop(state *GameState, scanner *bufio.Scanner) {
	for {
		if state.Player.HP <= 0 {
			fmt.Println("Vous etes mort.")
			return
		}
		if state.Monster.HP <= 0 {
			rewardVictory(state)
		}

		fmt.Printf("\nCombat: %s (%d/%d HP) vs %s (%d/%d HP)\n",
			state.Player.Name, state.Player.HP, state.Player.MaxHP,
			state.Monster.Name, state.Monster.HP, state.Monster.MaxHP,
		)
		fmt.Printf("Mana: %d/%d | Arme: %s | Ammo: %d | Laser: %d\n",
			state.Player.Mana, state.Player.MaxMana, equippedWeapon(state.Player), state.Player.Ammo, state.Player.LaserShots,
		)

		mode, dmg, _ := attackProfile(state.Player)
		fmt.Println("1) Attaque (" + mode + " " + strconv.Itoa(dmg) + " dmg)")
		fmt.Println("2) Boule de feu (" + strconv.Itoa(fireballDamage(state.Player)) + " dmg / 5 mana)")
		fmt.Println("3) Pouvoir magique")
		fmt.Println("4) Grenade")
		fmt.Println("5) Retour menu")
		fmt.Print("Action > ")

		action := readChoice(scanner)
		switch action {
		case "1":
			mode, dmg, consume := attackProfile(state.Player)
			if consume == "ammo" {
				state.Player.Ammo--
			}
			if consume == "laser" {
				state.Player.LaserShots--
			}
			state.Monster.HP -= dmg
			pushLog(state, fmt.Sprintf("%s inflige %d degats.", mode, dmg))
			triggerPlayerMagic(state, mode)
			if state.Monster.HP > 0 {
				enemyTurn(state)
			}
		case "2":
			if state.Player.Mana < 5 {
				pushLog(state, "Mana insuffisant pour Boule de feu.")
			} else {
				dmg := fireballDamage(state.Player)
				state.Player.Mana -= 5
				state.Monster.HP -= dmg
				pushLog(state, fmt.Sprintf("Boule de feu: %d degats.", dmg))
				triggerPlayerMagic(state, "Boule de feu")
				if state.Monster.HP > 0 {
					enemyTurn(state)
				}
			}
		case "3":
			if time.Now().Before(state.Player.MagicCooldownUntil) {
				remaining := time.Until(state.Player.MagicCooldownUntil).Round(time.Second)
				pushLog(state, "Pouvoir magique en recharge: "+remaining.String())
			} else {
				castPlayerMagicPower(state)
				if state.Monster.HP > 0 {
					enemyTurn(state)
				}
			}
		case "4":
			res := useItem(state, "Grenade")
			pushLog(state, res)
			if state.Monster.HP > 0 && state.Player.HP > 0 {
				enemyTurn(state)
			}
		case "5":
			return
		default:
			fmt.Println("Action invalide.")
		}

		for _, line := range state.CombatLog {
			fmt.Println("-", line)
		}
		if err := saveState(saveFilePath, *state); err != nil {
			fmt.Println("Avertissement: sauvegarde auto echouee:", err)
		}
	}
}

func enemyTurn(state *GameState) {
	state.Player.Turn++
	damage := enemyAttackDamage(state.Player, state.Monster)
	manaDrain := 0
	roll := rand.Float64()
	p := state.Monster.MagicPower

	if p == "Crachat toxique" && roll < 0.22 {
		damage += 6
		pushLog(state, state.Monster.Name+" utilise Crachat toxique (+6).")
	}
	if p == "Cri de guerre" && roll < 0.24 {
		damage += 10
		pushLog(state, state.Monster.Name+" utilise Cri de guerre (+10).")
	}
	if p == "Regeneration brute" && roll < 0.28 {
		heal := 12
		state.Monster.HP = minInt(state.Monster.MaxHP, state.Monster.HP+heal)
		pushLog(state, state.Monster.Name+" recupere 12 HP.")
	}
	if p == "Souffle draconique" && roll < 0.30 {
		damage += 14
		manaDrain = 3
		pushLog(state, state.Monster.Name+" souffle et gagne +14 degats.")
	}
	if p == "Flamme infernale" && roll < 0.30 {
		damage += 16
		manaDrain = 5
		pushLog(state, state.Monster.Name+" lance Flamme infernale (+16).")
	}
	if p == "Onde cataclysmique" && roll < 0.34 {
		damage += 20
		manaDrain = 7
		pushLog(state, state.Monster.Name+" lance Onde cataclysmique (+20).")
	}

	if state.Player.Turn%3 == 0 {
		damage *= 2
		pushLog(state, "Le monstre entre en rage: degats x2.")
	}

	state.Player.HP -= damage
	if state.Player.HP < 0 {
		state.Player.HP = 0
	}
	pushLog(state, fmt.Sprintf("%s inflige %d degats.", state.Monster.Name, damage))

	if manaDrain > 0 {
		state.Player.Mana = maxInt(0, state.Player.Mana-manaDrain)
		pushLog(state, fmt.Sprintf("Perte de mana: %d.", manaDrain))
	}
}

func triggerPlayerMagic(state *GameState, source string) {
	if state.Player.HP <= 0 || state.Monster.HP <= 0 {
		return
	}
	roll := rand.Float64()
	bonus := 0

	switch state.Player.MagicPower {
	case "Rage runique":
		if roll < 0.28 {
			bonus = 9
			state.Player.HP = minInt(state.Player.MaxHP, state.Player.HP+5)
			pushLog(state, "Rage runique: +9 degats et +5 HP.")
		}
	case "Aura tactique":
		if roll < 0.28 {
			bonus = 6
			state.Player.HP = minInt(state.Player.MaxHP, state.Player.HP+3)
			state.Player.Mana = minInt(state.Player.MaxMana, state.Player.Mana+4)
			pushLog(state, "Aura tactique: +6 degats, +3 HP, +4 mana.")
		}
	case "Surcharge arcane":
		if roll < 0.32 {
			bonus = 12
			state.Player.Mana = minInt(state.Player.MaxMana, state.Player.Mana+6)
			pushLog(state, "Surcharge arcane: +12 degats et +6 mana.")
		}
	}

	if bonus > 0 {
		state.Monster.HP -= bonus
		pushLog(state, fmt.Sprintf("Effet declenche apres %s.", source))
	}
}

func castPlayerMagicPower(state *GameState) {
	damage := 0
	message := ""
	switch state.Player.MagicPower {
	case "Rage runique":
		damage = 24
		state.Player.HP = minInt(state.Player.MaxHP, state.Player.HP+8)
		message = "Rage runique: 24 degats, +8 HP."
	case "Aura tactique":
		damage = 18
		state.Player.HP = minInt(state.Player.MaxHP, state.Player.HP+5)
		state.Player.Mana = minInt(state.Player.MaxMana, state.Player.Mana+10)
		message = "Aura tactique: 18 degats, +5 HP, +10 mana."
	case "Surcharge arcane":
		damage = 28
		state.Player.Mana = minInt(state.Player.MaxMana, state.Player.Mana+8)
		state.Player.LaserShots++
		message = "Surcharge arcane: 28 degats, +8 mana, +1 tir laser."
	default:
		damage = 16
		message = "Pouvoir mystique: 16 degats."
	}

	state.Monster.HP -= damage
	state.Player.MagicCooldownUntil = time.Now().Add(magicCooldown)
	pushLog(state, message)
}

func rewardVictory(state *GameState) {
	defeated := state.Monster.Name
	state.Player.Gold += 10
	gainXP(state, 5)
	state.Player.Turn = 0
	state.Monster = monsterForLevel(state.Player.Level)
	pushLog(state, fmt.Sprintf("Victoire contre %s. +10 or, +5 XP.", defeated))
}

func gainXP(state *GameState, amount int) {
	oldArena := minInt(state.Player.Level, 6)
	state.Player.Exp += amount
	for state.Player.Exp >= state.Player.MaxExp {
		state.Player.Exp -= state.Player.MaxExp
		state.Player.Level++
		state.Player.MaxExp += 10
		state.Player.MaxHP += 5
		state.Player.MaxMana += 2
		state.Player.HP = state.Player.MaxHP
		state.Player.Mana = state.Player.MaxMana

		newArena := minInt(state.Player.Level, 6)
		if newArena > oldArena {
			state.Player.Gold += 70
			state.Player.MaxMana += 8
			state.Player.Mana = state.Player.MaxMana
			pushLog(state, fmt.Sprintf("Nouvelle arene %d debloquee: +70 or, +8 mana max.", newArena))
			oldArena = newArena
		}
		pushLog(state, fmt.Sprintf("Niveau %d atteint.", state.Player.Level))
	}
}

func inventoryLoop(state *GameState, scanner *bufio.Scanner) {
	for {
		fmt.Println("\nInventaire:")
		if len(state.Player.Inventory) == 0 {
			fmt.Println("- vide")
		} else {
			for i, item := range state.Player.Inventory {
				fmt.Printf("%d) %s\n", i+1, item)
			}
		}
		fmt.Println("0) Retour")
		fmt.Print("Utiliser item (index) > ")

		raw := readChoice(scanner)
		if raw == "0" || raw == "" {
			return
		}
		idx, err := strconv.Atoi(raw)
		if err != nil || idx < 1 || idx > len(state.Player.Inventory) {
			fmt.Println("Index invalide.")
			continue
		}

		item := state.Player.Inventory[idx-1]
		message := useItem(state, item)
		fmt.Println(message)
		if err := saveState(saveFilePath, *state); err != nil {
			fmt.Println("Avertissement: sauvegarde auto echouee:", err)
		}
	}
}

func useItem(state *GameState, item string) string {
	switch item {
	case "Potion":
		if !consumeItem(state, "Potion") {
			return "Potion introuvable."
		}
		state.Player.HP = minInt(state.Player.MaxHP, state.Player.HP+consumablePotionHeal)
		return "Potion utilisee: +10 HP."
	case "Poison":
		if !consumeItem(state, "Poison") {
			return "Poison introuvable."
		}
		state.Player.HP = maxInt(0, state.Player.HP-5)
		return "Poison utilise: -5 HP."
	case "Livre":
		if !consumeItem(state, "Livre") {
			return "Livre introuvable."
		}
		if !hasSkill(state.Player, "Boule de feu") {
			state.Player.Skills = append(state.Player.Skills, "Boule de feu")
			return "Nouvelle competence: Boule de feu."
		}
		return "Livre lu: aucune competence supplementaire."
	case "Casque":
		if !consumeItem(state, "Casque") {
			return "Casque introuvable."
		}
		state.Player.MaxHP += 5
		state.Player.HP = minInt(state.Player.MaxHP, state.Player.HP+5)
		state.Player.Equip.Head = "Casque"
		return "Casque equipe: +5 HP max."
	case "Armure":
		if !consumeItem(state, "Armure") {
			return "Armure introuvable."
		}
		state.Player.MaxHP += 10
		state.Player.HP = minInt(state.Player.MaxHP, state.Player.HP+10)
		state.Player.Equip.Body = "Armure"
		return "Armure equipee: +10 HP max."
	case "Bottes":
		if !consumeItem(state, "Bottes") {
			return "Bottes introuvables."
		}
		state.Player.MaxHP += 5
		state.Player.HP = minInt(state.Player.MaxHP, state.Player.HP+5)
		state.Player.Equip.Feet = "Bottes"
		return "Bottes equipees: +5 HP max."
	case "Pistolet", "Epee", "Laser", "Lance-roquettes":
		state.Player.Equip.Weapon = item
		return item + " equipe."
	case "Munitions":
		if !consumeItem(state, "Munitions") {
			return "Munitions introuvables."
		}
		state.Player.Ammo += 6
		return "Munitions chargees: +6."
	case "Batterie":
		if !consumeItem(state, "Batterie") {
			return "Batterie introuvable."
		}
		state.Player.LaserShots += 2
		return "Batterie utilisee: +2 tirs laser."
	case "Grenade":
		if !consumeItem(state, "Grenade") {
			return "Grenade introuvable."
		}
		state.Monster.HP -= grenadeDamage
		if state.Monster.HP <= 0 {
			rewardVictory(state)
			return "Grenade decisive: monstre vaincu."
		}
		return fmt.Sprintf("Grenade lancee: %d degats.", grenadeDamage)
	default:
		return "Item inconnu."
	}
}

func consumeItem(state *GameState, item string) bool {
	for i, entry := range state.Player.Inventory {
		if entry == item {
			state.Player.Inventory = append(state.Player.Inventory[:i], state.Player.Inventory[i+1:]...)
			return true
		}
	}
	return false
}

func shopLoop(state *GameState, scanner *bufio.Scanner) {
	items := []string{"Potion", "Poison", "Livre", "Casque", "Armure", "Bottes", "Pistolet", "Epee", "Laser", "Lance-roquettes", "Munitions", "Batterie", "Grenade"}
	for {
		fmt.Printf("\nMarchand (or: %d)\n", state.Player.Gold)
		for i, item := range items {
			fmt.Printf("%d) %-16s %d or\n", i+1, item, shopPrices[item])
		}
		fmt.Println("0) Retour")
		fmt.Print("Acheter > ")

		raw := readChoice(scanner)
		if raw == "0" || raw == "" {
			return
		}
		idx, err := strconv.Atoi(raw)
		if err != nil || idx < 1 || idx > len(items) {
			fmt.Println("Index invalide.")
			continue
		}
		item := items[idx-1]
		cost := shopPrices[item]
		if state.Player.Gold < cost {
			fmt.Println("Pas assez d'or.")
			continue
		}
		state.Player.Gold -= cost
		state.Player.Inventory = append(state.Player.Inventory, item)
		fmt.Printf("Achat: %s pour %d or.\n", item, cost)
		if err := saveState(saveFilePath, *state); err != nil {
			fmt.Println("Avertissement: sauvegarde auto echouee:", err)
		}
	}
}

func chooseCharacter(state *GameState, scanner *bufio.Scanner) {
	fmt.Println("\nPersonnages:")
	for i, p := range characterPresets {
		fmt.Printf("%d) %s (%s) HP:%d Mana:%d Pouvoir:%s\n", i+1, p.Name, p.Class, p.MaxHP, p.MaxMana, p.MagicPower)
	}
	fmt.Print("Selection > ")
	raw := readChoice(scanner)
	idx, err := strconv.Atoi(raw)
	if err != nil || idx < 1 || idx > len(characterPresets) {
		fmt.Println("Selection invalide.")
		return
	}
	next := characterPresets[idx-1]
	*state = newState(next.ID)
	fmt.Printf("Personnage actif: %s.\n", next.Name)
}

func changeName(state *GameState, scanner *bufio.Scanner) {
	fmt.Print("Nouveau nom > ")
	name := strings.TrimSpace(readChoice(scanner))
	if name == "" {
		fmt.Println("Nom vide ignore.")
		return
	}
	if len(name) > 24 {
		name = name[:24]
	}
	state.Player.Name = name
	fmt.Println("Nom mis a jour.")
}

func handleDeath(state *GameState, scanner *bufio.Scanner) {
	now := time.Now()
	rc := &state.RestartControl
	if !rc.CooldownUntil.IsZero() && now.After(rc.CooldownUntil) {
		rc.Used = 0
		rc.CooldownUntil = time.Time{}
	}

	if !rc.CooldownUntil.IsZero() && now.Before(rc.CooldownUntil) {
		fmt.Printf("Vous etes mort. Recommencer possible dans %s.\n", time.Until(rc.CooldownUntil).Round(time.Second))
		return
	}

	if rc.Used >= restartLimit {
		rc.CooldownUntil = now.Add(restartCooldown)
		fmt.Printf("Limite de %d tentatives atteinte. Cooldown: %s.\n", restartLimit, restartCooldown)
		return
	}

	fmt.Printf("Vous etes mort. Recommencer ? (%d/%d tentatives restantes) [o/n] > ", restartLimit-rc.Used, restartLimit)
	ans := strings.ToLower(strings.TrimSpace(readChoice(scanner)))
	if ans != "o" && ans != "oui" && ans != "y" {
		return
	}

	rc.Used++
	if rc.Used >= restartLimit {
		rc.CooldownUntil = now.Add(restartCooldown)
	}

	characterID := state.Player.CharacterID
	if characterID == "" {
		characterID = characterPresets[0].ID
	}
	fresh := playerFromCharacter(characterID)
	state.Player = fresh
	state.Monster = monsterForLevel(fresh.Level)
	state.CombatLog = []string{"Nouvelle tentative lancee."}
}

func printStatus(state GameState) {
	fmt.Printf("\n=== %s (%s) ===\n", state.Player.Name, state.Player.Class)
	fmt.Printf("Niveau %d | XP %d/%d | Or %d\n", state.Player.Level, state.Player.Exp, state.Player.MaxExp, state.Player.Gold)
	fmt.Printf("HP %d/%d | Mana %d/%d\n", state.Player.HP, state.Player.MaxHP, state.Player.Mana, state.Player.MaxMana)
	fmt.Printf("Arme %s | Ammo %d | Laser %d\n", equippedWeapon(state.Player), state.Player.Ammo, state.Player.LaserShots)
	fmt.Printf("Monstre actuel: %s (%d/%d HP)\n", state.Monster.Name, state.Monster.HP, state.Monster.MaxHP)
}

func equippedWeapon(p Player) string {
	if p.Equip.Weapon == "" {
		return "Coup de poing"
	}
	return p.Equip.Weapon
}

func pushLog(state *GameState, line string) {
	state.CombatLog = append(state.CombatLog, line)
	if len(state.CombatLog) > 8 {
		state.CombatLog = state.CombatLog[len(state.CombatLog)-8:]
	}
}

func hasSkill(p Player, wanted string) bool {
	for _, skill := range p.Skills {
		if skill == wanted {
			return true
		}
	}
	return false
}

func readChoice(scanner *bufio.Scanner) string {
	if !scanner.Scan() {
		return ""
	}
	return strings.TrimSpace(scanner.Text())
}

func saveState(path string, state GameState) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := gob.NewEncoder(file)
	return encoder.Encode(state)
}

func loadState(path string) (GameState, error) {
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return newState(characterPresets[0].ID), nil
		}
		return GameState{}, err
	}
	defer file.Close()

	var state GameState
	decoder := gob.NewDecoder(file)
	if err := decoder.Decode(&state); err != nil {
		if errors.Is(err, io.EOF) {
			return newState(characterPresets[0].ID), nil
		}
		return GameState{}, err
	}

	normalizeState(&state)
	return state, nil
}

func mergePlayer(base Player, loaded Player) Player {
	out := base
	if loaded.CharacterID != "" {
		out.CharacterID = loaded.CharacterID
	}
	if loaded.Name != "" {
		out.Name = loaded.Name
	}
	if loaded.Class != "" {
		out.Class = loaded.Class
	}
	if loaded.MagicPower != "" {
		out.MagicPower = loaded.MagicPower
	}
	if loaded.Level > 0 {
		out.Level = loaded.Level
	}
	if loaded.MaxHP > 0 {
		out.MaxHP = loaded.MaxHP
	}
	if loaded.HP > 0 {
		out.HP = loaded.HP
	}
	if loaded.MaxMana > 0 {
		out.MaxMana = loaded.MaxMana
	}
	if loaded.Mana >= 0 {
		out.Mana = loaded.Mana
	}
	if loaded.Gold >= 0 {
		out.Gold = loaded.Gold
	}
	if loaded.Exp >= 0 {
		out.Exp = loaded.Exp
	}
	if loaded.MaxExp > 0 {
		out.MaxExp = loaded.MaxExp
	}
	if loaded.Turn >= 0 {
		out.Turn = loaded.Turn
	}
	if len(loaded.Inventory) > 0 {
		out.Inventory = loaded.Inventory
	}
	if len(loaded.Skills) > 0 {
		out.Skills = loaded.Skills
	}
	out.Ammo = loaded.Ammo
	out.LaserShots = loaded.LaserShots
	if !loaded.MagicCooldownUntil.IsZero() {
		out.MagicCooldownUntil = loaded.MagicCooldownUntil
	}
	if loaded.Equip.Head != "" {
		out.Equip.Head = loaded.Equip.Head
	}
	if loaded.Equip.Body != "" {
		out.Equip.Body = loaded.Equip.Body
	}
	if loaded.Equip.Feet != "" {
		out.Equip.Feet = loaded.Equip.Feet
	}
	if loaded.Equip.Weapon != "" {
		out.Equip.Weapon = loaded.Equip.Weapon
	}
	return out
}

func normalizeState(state *GameState) {
	if state.Player.Level < 1 {
		state.Player.Level = 1
	}
	if state.Player.MaxExp <= 0 {
		state.Player.MaxExp = 10
	}
	if state.Player.MaxHP <= 0 {
		state.Player.MaxHP = 20
	}
	if state.Player.MaxMana <= 0 {
		state.Player.MaxMana = 10
	}
	state.Player.HP = minInt(state.Player.MaxHP, maxInt(0, state.Player.HP))
	state.Player.Mana = minInt(state.Player.MaxMana, maxInt(0, state.Player.Mana))
	if state.Player.Inventory == nil {
		state.Player.Inventory = []string{}
	}
	if state.Player.Skills == nil {
		state.Player.Skills = []string{"Coup de poing"}
	}
	if state.Player.MagicPower == "" {
		state.Player.MagicPower = "Etincelle mystique"
	}

	if state.Monster.Name == "" {
		state.Monster = monsterForLevel(state.Player.Level)
	} else {
		state.Monster.MagicPower = monsterMagic(state.Monster.Name)
		if state.Monster.MaxHP <= 0 {
			state.Monster = monsterForLevel(state.Player.Level)
		}
		if state.Monster.HP < 0 {
			state.Monster.HP = 0
		}
	}

	if state.CombatLog == nil {
		state.CombatLog = []string{}
	}
	if state.Theme == "" {
		state.Theme = "neonwave"
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
