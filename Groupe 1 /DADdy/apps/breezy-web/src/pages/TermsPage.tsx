import { Link } from 'react-router-dom'
import { ArrowLeft } from 'lucide-react'
import { BreezyLogo } from '@/components/BreezyLogo'
import { LanguageSwitcher } from '@/components/LanguageSwitcher'
import { ThemeToggle } from '@/components/ThemeToggle'
import { useLanguage } from '@/hooks/useLanguage'
import { TERMS_VERSION } from '@/lib/terms'

const content = {
  fr: {
    title: 'Contrat de licence utilisateur final',
    updated: `Version ${TERMS_VERSION}`,
    back: "Retour à l'inscription",
    intro:
      "Le présent contrat encadre l'accès et l'utilisation de Breezy. En créant un compte, vous acceptez cette version du contrat.",
    sections: [
      ['1. Licence d’utilisation', "Breezy vous accorde une licence personnelle, limitée, non exclusive, non transférable et révocable pour utiliser le service conformément au présent contrat."],
      ['2. Compte', "Vous devez fournir des informations exactes, protéger vos identifiants et nous signaler tout accès non autorisé. Vous êtes responsable de l'activité réalisée depuis votre compte."],
      ['3. Vos contenus', "Vous conservez vos droits sur les contenus publiés. Vous accordez à Breezy les droits techniques nécessaires pour les héberger, les reproduire et les afficher uniquement afin de fournir le service."],
      ['4. Utilisations interdites', "Il est interdit de publier des contenus illicites, menaçants ou contrefaisants, d'usurper une identité, de harceler d'autres personnes, de contourner les protections du service ou d'en perturber le fonctionnement."],
      ['5. Modération et résiliation', "Breezy peut retirer un contenu ou suspendre un compte qui enfreint ce contrat ou la loi. Vous pouvez cesser d'utiliser le service à tout moment."],
      ['6. Disponibilité', "Le service est fourni en l'état et peut évoluer ou être interrompu. Dans les limites autorisées par la loi, Breezy ne garantit pas une disponibilité permanente ni l'absence totale d'erreurs."],
      ['7. Évolution du contrat', "La version applicable lors de votre inscription est enregistrée avec votre acceptation. Toute nouvelle version sera publiée sur cette page avec sa date d'entrée en vigueur."],
    ],
  },
  en: {
    title: 'End-user license agreement',
    updated: `Version ${TERMS_VERSION}`,
    back: 'Back to registration',
    intro:
      'This agreement governs access to and use of Breezy. By creating an account, you accept this version of the agreement.',
    sections: [
      ['1. License to use', 'Breezy grants you a personal, limited, non-exclusive, non-transferable, and revocable license to use the service under this agreement.'],
      ['2. Account', 'You must provide accurate information, protect your credentials, and report unauthorized access. You are responsible for activity performed through your account.'],
      ['3. Your content', 'You retain your rights to posted content. You grant Breezy the technical rights needed to host, reproduce, and display it solely to provide the service.'],
      ['4. Prohibited use', 'You may not post unlawful, threatening, or infringing content, impersonate others, harass people, bypass service protections, or disrupt operation of the service.'],
      ['5. Moderation and termination', 'Breezy may remove content or suspend accounts that violate this agreement or the law. You may stop using the service at any time.'],
      ['6. Availability', 'The service is provided as available and may change or be interrupted. To the extent permitted by law, Breezy does not guarantee uninterrupted or error-free operation.'],
      ['7. Agreement updates', 'The version applicable when you register is recorded with your acceptance. New versions will be published on this page with their effective date.'],
    ],
  },
}

export function TermsPage() {
  const { locale } = useLanguage()
  const terms = locale === 'fr' ? content.fr : content.en

  return (
    <main className="relative z-10 min-h-screen px-5 py-8 sm:px-8">
      <div className="mx-auto max-w-3xl">
        <header className="mb-8 flex items-center justify-between gap-4">
          <BreezyLogo variant="text" size={26} />
          <div className="flex items-center gap-2">
            <ThemeToggle />
            <LanguageSwitcher />
          </div>
        </header>

        <article className="rounded-2xl border border-border bg-card p-6 shadow-lg shadow-black/8 ring-1 ring-primary/12 sm:p-10">
          <Link
            to="/register"
            className="mb-6 inline-flex items-center gap-2 text-sm font-medium text-primary hover:text-primary/80"
          >
            <ArrowLeft className="size-4" />
            {terms.back}
          </Link>

          <h1 className="text-3xl font-bold text-foreground">{terms.title}</h1>
          <p className="mt-2 text-xs font-medium uppercase tracking-wider text-muted-foreground">
            {terms.updated}
          </p>
          <p className="mt-6 text-sm leading-7 text-muted-foreground">{terms.intro}</p>

          <div className="mt-8 space-y-7">
            {terms.sections.map(([title, body]) => (
              <section key={title}>
                <h2 className="text-base font-semibold text-foreground">{title}</h2>
                <p className="mt-2 text-sm leading-7 text-muted-foreground">{body}</p>
              </section>
            ))}
          </div>
        </article>
      </div>
    </main>
  )
}
