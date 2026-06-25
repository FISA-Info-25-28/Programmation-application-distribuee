import { useQuery } from '@tanstack/react-query'
import { ExternalLink } from 'lucide-react'

const URL_RE = /https?:\/\/[^\s<>"{}|\\^`[\]]{4,}/g

function extractFirstUrl(content: string): string | null {
  const matches = content.match(URL_RE)
  return matches?.[0] ?? null
}

interface OGData {
  title: string | null
  description: string | null
  image: { url: string } | null
  publisher: string | null
  url: string
}

async function fetchOG(url: string): Promise<OGData> {
  const res = await fetch(`https://api.microlink.io?url=${encodeURIComponent(url)}`)
  if (!res.ok) throw new Error('fetch failed')
  const json = await res.json()
  if (json.status !== 'success') throw new Error('og failed')
  return {
    title: json.data.title ?? null,
    description: json.data.description ?? null,
    image: json.data.image ?? null,
    publisher: json.data.publisher ?? null,
    url: json.data.url ?? url,
  }
}

function hostname(url: string) {
  try { return new URL(url).hostname.replace(/^www\./, '') } catch { return url }
}

export function LinkPreview({ content }: { content: string }) {
  const url = extractFirstUrl(content)

  const { data, isError } = useQuery({
    queryKey: ['link-preview', url],
    queryFn: () => fetchOG(url!),
    enabled: !!url,
    staleTime: 24 * 60 * 60 * 1000,
    retry: false,
    gcTime: 24 * 60 * 60 * 1000,
  })

  if (!url || isError || !data?.title) return null

  return (
    <a
      href={url}
      target="_blank"
      rel="noopener noreferrer"
      onClick={(e) => e.stopPropagation()}
      className="mt-2.5 flex flex-col rounded-xl border border-border overflow-hidden hover:bg-accent/40 transition-colors group"
    >
      {data.image?.url && (
        <img
          src={data.image.url}
          alt={data.title ?? ''}
          className="w-full h-40 object-cover"
          onError={(e) => { (e.target as HTMLImageElement).style.display = 'none' }}
        />
      )}
      <div className="px-3 py-2.5 flex items-start justify-between gap-2">
        <div className="min-w-0">
          <p className="text-[11px] text-muted-foreground truncate">{data.publisher ?? hostname(url)}</p>
          <p className="text-sm font-medium text-foreground mt-0.5 line-clamp-2 leading-snug">{data.title}</p>
          {data.description && (
            <p className="text-xs text-muted-foreground mt-0.5 line-clamp-2">{data.description}</p>
          )}
        </div>
        <ExternalLink size={13} className="shrink-0 mt-1 text-muted-foreground group-hover:text-foreground transition-colors" />
      </div>
    </a>
  )
}
