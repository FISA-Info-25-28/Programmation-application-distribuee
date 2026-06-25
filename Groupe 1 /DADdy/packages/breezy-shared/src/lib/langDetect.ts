// Returns a BCP47-style language code if the language can be identified with
// reasonable confidence, or null if uncertain.
// Non-Latin scripts are detected by Unicode range (accurate).
// Latin scripts use stop-word scoring (heuristic, works well for 5+ word texts).

const STOP_WORDS: Record<string, string[]> = {
  fr: ['le', 'la', 'les', 'de', 'du', 'des', 'un', 'une', 'est', 'et', 'je', 'tu', 'il',
       'nous', 'vous', 'ils', 'ce', 'se', 'ne', 'pas', 'que', 'qui', 'au', 'aux', 'sur',
       'dans', 'avec', 'par', 'mais', 'ou', 'donc', 'or', 'ni', 'car'],
  en: ['the', 'and', 'this', 'that', 'with', 'for', 'you', 'are', 'have', 'was', 'not',
       'but', 'from', 'they', 'she', 'his', 'her', 'its', 'our', 'their', 'will', 'been',
       'has', 'had', 'would', 'could', 'should', 'may', 'might'],
  es: ['el', 'los', 'del', 'una', 'para', 'con', 'por', 'esto', 'pero', 'más', 'ser',
       'como', 'sus', 'entre', 'cuando', 'también', 'todo', 'esta', 'ese', 'hay'],
  de: ['der', 'die', 'das', 'und', 'ist', 'ein', 'nicht', 'sich', 'auch', 'dem', 'den',
       'mit', 'auf', 'von', 'bei', 'nach', 'aus', 'vor', 'über', 'oder', 'aber', 'wenn'],
  pt: ['não', 'com', 'para', 'mais', 'uma', 'por', 'são', 'mas', 'isso', 'ela', 'ele',
       'nos', 'dos', 'das', 'seu', 'sua', 'essa', 'esse', 'como', 'quando', 'também'],
  it: ['che', 'non', 'con', 'per', 'sono', 'anche', 'questo', 'della', 'degli', 'dei',
       'una', 'nel', 'nel', 'dal', 'alla', 'alle', 'tra', 'fra', 'come', 'quando', 'già'],
}

export function detectLang(text: string): string | null {
  if (!text || text.trim().length < 8) return null

  // Non-Latin scripts — character range detection (reliable)
  if (/[؀-ۿ]/.test(text)) return 'ar'
  if (/[֐-׿]/.test(text)) return 'iw' // Google uses iw for Hebrew
  if (/[Ѐ-ӿ]/.test(text)) return 'ru'
  if (/[぀-ヿ一-鿿㐀-䶿]/.test(text)) return 'ja'
  if (/[가-힯]/.test(text)) return 'ko'

  // Latin scripts — stop-word frequency scoring
  const words = text.toLowerCase().match(/\b[a-zàâäéèêëîïôùûüçœæãõáíóúñ]+\b/g) ?? []
  if (words.length < 3) return null

  const scores: Record<string, number> = {}
  for (const [lang, markers] of Object.entries(STOP_WORDS)) {
    scores[lang] = words.filter((w) => markers.includes(w)).length
  }

  const sorted = Object.entries(scores).sort((a, b) => b[1] - a[1])
  const [first, second] = sorted

  // Need at least 1 hit and a clear lead over the runner-up
  if (!first || first[1] === 0) return null
  if (second && first[1] === second[1]) return null

  return first[0]
}
