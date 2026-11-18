package domain

import (
	"strings"
	"unicode/utf8"
)

var Topics = []struct{ ID, Question string }{
	{"housing", "Где вы проводили ночи на этой неделе и какие планы на ближайшие дни?"},
	{"expenses", "Какие расходы удалось покрыть, а что пришлось отложить?"},
	{"work", "Чем вы занимались на этой неделе? Что хотелось бы изменить в работе?"},
	{"connection", "С кем удалось поговорить так, как вам хотелось? Какого общения не хватило?"},
	{"daily", "В какие моменты было легче справляться с повседневными делами? Что помогало?"},
	{"priority", "Что требует внимания уже сегодня? Какую поддержку вы бы выбрали первой?"},
}

var Variants = map[string]map[string]string{
	"housing":    {"resources": "Что помогало вам находить место для отдыха и ночлега?", "obstacles": "Что было труднее всего в вопросе жилья на этой неделе?"},
	"expenses":   {"resources": "Что помогло покрыть основные расходы на этой неделе?", "obstacles": "Какие покупки или платежи было труднее всего запланировать?"},
	"work":       {"resources": "Какие навыки и занятия сейчас помогают вам зарабатывать или искать работу?", "obstacles": "Что мешало заниматься работой, которая вам подходит?"},
	"connection": {"resources": "К кому вы могли обратиться и какую поддержку получили?", "obstacles": "Когда хотелось поговорить, что мешало найти собеседника?"},
	"daily":      {"resources": "Что давало вам силы справляться с обычными делами?", "obstacles": "Какие повседневные дела потребовали больше сил, чем обычно?"},
	"priority":   {"resources": "На какую поддержку вы хотели бы опереться в первую очередь?", "obstacles": "Какую одну трудность вы хотели бы решить первой?"},
}

func (v Submission) ValidInterview() bool {
	if !v.Consent || v.QuestionnaireID != "saya-week" || v.Version != 2 || utf8.RuneCountInString(v.Summary) > 1800 {
		return false
	}
	for key := range v.Answers {
		if key != "immediate_threat" {
			return false
		}
	}
	if len(v.Ratings)+len(v.Responses)+len(v.Needs) == 0 && strings.TrimSpace(v.Summary) == "" && !v.Answers["immediate_threat"] {
		return false
	}
	allowed := map[string]bool{}
	for _, t := range Topics {
		allowed[t.ID] = true
	}
	for key, focus := range v.QuestionFocus {
		if !allowed[key] || (focus != "routine" && Variants[key][focus] == "") {
			return false
		}
	}
	for key, value := range v.Responses {
		if !allowed[key] || strings.TrimSpace(value) == "" || utf8.RuneCountInString(value) > 800 {
			return false
		}
	}
	for key, value := range v.Ratings {
		if !allowed[key] || key == "priority" || value < 0 || value > 4 {
			return false
		}
	}
	needs := map[string]bool{"housing": true, "jobs": true, "psychologist": true, "social": true, "financial": true}
	for _, key := range v.Needs {
		if !needs[key] {
			return false
		}
		delete(needs, key)
		if key == "financial" {
			rating, ok := v.Ratings["expenses"]
			if !ok || rating > 1 {
				return false
			}
		}
	}
	return true
}
