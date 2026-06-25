const en = {
  nav: 'Bookmarks',
  title: 'Bookmarks',
  subtitle: 'Posts you saved for later',
  empty: 'You have no bookmarks yet.',
  bookmark: 'Bookmark',
  removeBookmark: 'Remove bookmark',
}

const fr = {
  nav: 'Signets',
  title: 'Signets',
  subtitle: 'Les posts que vous avez enregistrés',
  empty: "Vous n'avez aucun signet pour l'instant.",
  bookmark: 'Enregistrer',
  removeBookmark: 'Retirer le signet',
}

const it = {
  nav: 'Segnalibri',
  title: 'Segnalibri',
  subtitle: 'I post che hai salvato per dopo',
  empty: 'Non hai ancora segnalibri.',
  bookmark: 'Salva',
  removeBookmark: 'Rimuovi dai segnalibri',
}

const ja = {
  nav: 'ブックマーク',
  title: 'ブックマーク',
  subtitle: '後で読むために保存した投稿',
  empty: 'まだブックマークがありません。',
  bookmark: 'ブックマーク',
  removeBookmark: 'ブックマークを削除',
}

const ko = {
  nav: '북마크',
  title: '북마크',
  subtitle: '나중에 읽기 위해 저장한 게시물',
  empty: '아직 북마크가 없습니다.',
  bookmark: '북마크',
  removeBookmark: '북마크 제거',
}

const es = {
  nav: 'Guardados',
  title: 'Guardados',
  subtitle: 'Publicaciones que guardaste para más tarde',
  empty: 'No tienes publicaciones guardadas aún.',
  bookmark: 'Guardar',
  removeBookmark: 'Quitar de guardados',
}

const ar = {
  nav: 'المحفوظات',
  title: 'المحفوظات',
  subtitle: 'المنشورات التي حفظتها لوقت لاحق',
  empty: 'ليس لديك أي منشورات محفوظة حتى الآن.',
  bookmark: 'حفظ',
  removeBookmark: 'إزالة من المحفوظات',
}

const he = {
  nav: 'סימניות',
  title: 'סימניות',
  subtitle: 'פוסטים ששמרת לאחר כך',
  empty: 'אין לך סימניות עדיין.',
  bookmark: 'שמור',
  removeBookmark: 'הסר סימנייה',
}

const ru = {
  nav: 'Закладки',
  title: 'Закладки',
  subtitle: 'Посты, которые вы сохранили на потом',
  empty: 'У вас пока нет закладок.',
  bookmark: 'Сохранить',
  removeBookmark: 'Убрать из закладок',
}

const de = {
  nav: 'Lesezeichen',
  title: 'Lesezeichen',
  subtitle: 'Beiträge, die du für später gespeichert hast',
  empty: 'Du hast noch keine Lesezeichen.',
  bookmark: 'Speichern',
  removeBookmark: 'Lesezeichen entfernen',
}

const pt = {
  nav: 'Salvos',
  title: 'Salvos',
  subtitle: 'Publicações que você salvou para depois',
  empty: 'Você ainda não tem publicações salvas.',
  bookmark: 'Salvar',
  removeBookmark: 'Remover dos salvos',
}

export type BookmarkStrings = typeof en

const map: Record<string, BookmarkStrings> = { fr, en, it, ja, ko, es, ar, he, ru, de, pt }

export function bookmarksT(locale: string): BookmarkStrings {
  return map[locale] ?? en
}
