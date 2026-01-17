package simcfg

import (
	"sort"
	"strconv"
	"strings"
)

var statKeys = []string{
	"n/a",
	"def%",
	"def",
	"hp",
	"hp%",
	"atk",
	"atk%",
	"er",
	"em",
	"cr",
	"cd",
	"heal",
	"pyro%",
	"hydro%",
	"cryo%",
	"electro%",
	"anemo%",
	"geo%",
	"dendro%",
	"phys%",
	"atkspd%",
	"dmg%",
}

func RenderSimConfig(chars []SimChar) string {
	var b strings.Builder
	for i, c := range chars {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(renderChar(c))
	}
	return b.String()
}

func renderChar(c SimChar) string {
	var b strings.Builder

	b.WriteString(c.Name)
	b.WriteString(" char lvl=")
	b.WriteString(strconv.Itoa(c.Level))
	b.WriteString("/")
	b.WriteString(strconv.Itoa(c.MaxLevel))
	b.WriteString(" cons=")
	b.WriteString(strconv.Itoa(c.Cons))
	b.WriteString(" talent=")
	b.WriteString(strconv.Itoa(c.Talents.Attack))
	b.WriteString(",")
	b.WriteString(strconv.Itoa(c.Talents.Skill))
	b.WriteString(",")
	b.WriteString(strconv.Itoa(c.Talents.Burst))
	b.WriteString(";\n")

	b.WriteString(c.Name)
	b.WriteString(" add weapon=\"")
	b.WriteString(c.Weapon.Name)
	b.WriteString("\" refine=")
	b.WriteString(strconv.Itoa(c.Weapon.Refine))
	b.WriteString(" lvl=")
	b.WriteString(strconv.Itoa(c.Weapon.Level))
	b.WriteString("/")
	b.WriteString(strconv.Itoa(c.Weapon.MaxLevel))
	b.WriteString(";\n")

	setKeys := make([]string, 0, len(c.Sets))
	for k := range c.Sets {
		if c.Sets[k] > 0 {
			setKeys = append(setKeys, k)
		}
	}
	sort.Strings(setKeys)
	for _, k := range setKeys {
		b.WriteString(c.Name)
		b.WriteString(" add set=\"")
		b.WriteString(k)
		b.WriteString("\" count=")
		b.WriteString(strconv.Itoa(c.Sets[k]))
		b.WriteString(";\n")
	}

	if line := renderStatsLine(c.Name, c.Main); line != "" {
		b.WriteString(line)
		b.WriteString(" #main\n")
	}
	if line := renderStatsLine(c.Name, c.Subs); line != "" {
		b.WriteString(line)
		b.WriteString("\n")
	}

	return b.String()
}

func renderStatsLine(name string, stats []float64) string {
	if len(stats) == 0 {
		return ""
	}

	count := 0
	var b strings.Builder
	b.WriteString(name)
	b.WriteString(" add stats")

	for i, v := range stats {
		if i >= len(statKeys) {
			break
		}
		if v == 0 {
			continue
		}
		key := statKeys[i]
		if key == "n/a" {
			continue
		}
		count++
		b.WriteString(" ")
		b.WriteString(key)
		b.WriteString("=")
		b.WriteString(strconv.FormatFloat(v, 'g', -1, 64))
	}

	if count == 0 {
		return ""
	}
	b.WriteString(";")
	return b.String()
}
