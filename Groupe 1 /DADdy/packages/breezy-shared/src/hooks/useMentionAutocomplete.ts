import { useState, useCallback, useEffect, useRef } from 'react'
import { useQuery } from '@tanstack/react-query'
import { searchUsers } from '../api/users'
import type { User, MentionedUser } from '../types/api'

interface MentionAutocomplete {
  mentionActive: boolean
  suggestions: User[]
  isLoading: boolean
  dropdownTop: number
  onContentChange: (value: string, cursorPos: number) => void
  selectUser: (user: User) => string
  mentions: MentionedUser[]
  resetMentions: () => void
}

// Détecte un @partial juste avant le curseur
const MENTION_RE = /@([a-zA-Z0-9_]{0,50})$/

export function useMentionAutocomplete(
  content: string,
  setContent: (v: string) => void,
): MentionAutocomplete {
  const [rawQuery, setRawQuery] = useState('')
  const [debouncedQuery, setDebouncedQuery] = useState('')
  const [mentionActive, setMentionActive] = useState(false)
  const [mentions, setMentions] = useState<MentionedUser[]>([])
  const [dropdownTop, setDropdownTop] = useState(0)
  // Position dans content où commence @partial (pour le remplacement)
  const matchStartRef = useRef<number>(-1)

  // line-height de text-sm leading-relaxed ≈ 22px
  const LINE_HEIGHT = 22

  // Debounce 150ms
  useEffect(() => {
    const t = setTimeout(() => setDebouncedQuery(rawQuery), 150)
    return () => clearTimeout(t)
  }, [rawQuery])

  const { data, isFetching } = useQuery({
    queryKey: ['mention-search', debouncedQuery],
    queryFn: () => searchUsers(debouncedQuery, 1, 5),
    enabled: mentionActive && debouncedQuery.length > 0,
    staleTime: 0,
  })

  const suggestions = data?.data ?? []

  const onContentChange = useCallback((value: string, cursorPos: number) => {
    const before = value.slice(0, cursorPos)
    const match = before.match(MENTION_RE)
    if (match) {
      const atPos = cursorPos - match[0].length
      matchStartRef.current = atPos
      // Nombre de lignes avant le @ → décalage vertical du dropdown
      const linesBefore = value.slice(0, atPos).split('\n').length - 1
      setDropdownTop((linesBefore + 1) * LINE_HEIGHT + 6)
      setRawQuery(match[1])
      setMentionActive(true)
    } else {
      setRawQuery('')
      setMentionActive(false)
      matchStartRef.current = -1
    }
  }, [LINE_HEIGHT])

  // Sélectionne un utilisateur : remplace @partial par @username dans le contenu
  const selectUser = useCallback((user: User): string => {
    const start = matchStartRef.current
    if (start === -1) return content
    const before = content.slice(0, start)
    const after = content.slice(start).replace(MENTION_RE, '')
    const newContent = `${before}@${user.username} ${after}`
    setContent(newContent)
    setMentionActive(false)
    setRawQuery('')
    matchStartRef.current = -1
    setMentions((prev) => {
      if (prev.find((m) => m.id === user.id)) return prev
      return [...prev, { id: user.id, username: user.username }]
    })
    return newContent
  }, [content, setContent])

  const resetMentions = useCallback(() => setMentions([]), [])

  return {
    mentionActive,
    suggestions,
    isLoading: isFetching,
    dropdownTop,
    onContentChange,
    selectUser,
    mentions,
    resetMentions,
  }
}
