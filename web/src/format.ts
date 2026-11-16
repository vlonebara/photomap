const pluralRules = new Intl.PluralRules('ru')

export function countLabel(count: number, forms: [string, string, string]) {
  const category = pluralRules.select(count)
  return `${count} ${forms[category === 'one' ? 0 : category === 'few' ? 1 : 2]}`
}

export function photoDate(year: number, endYear?: number) {
  return endYear && endYear !== year ? `${year}–${endYear}` : String(year)
}
