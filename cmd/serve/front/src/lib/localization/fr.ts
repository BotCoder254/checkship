import type { AppTranslations, Locale } from '#lib/localization/index.ts';
import routes from '#lib/path.ts';

export default {
	code: 'fr',
	displayName: 'Français',
	translations: {
		// Authentication
		'auth.signin.title': 'Connexion',
		'auth.signin.description':
			'Remplissez le formulaire ci-dessous pour accéder au tableau de bord.',
		'auth.tagline': 'Plateforme de déploiement auto-hébergée, sans prise de tête.',
		// App
		'app.no_targets': 'Aucune cible trouvée',
		'app.no_targets.description': `Vous avez besoin d'au moins une cible pour pouvoir déployer votre application. Dirigez-vous vers la <a href="${routes.createTarget}">page de création</a> pour en créer une.`,
		'app.not_found':
			"Il semblerait que l'application que vous recherchez n'existe pas. Retournez à la",
		'app.not_found.cta': "page d'accueil",
		'app.blankslate': `Aucune application pour le moment. <br />Les applications représentent les <strong>services que vous souhaitez déployer</strong> sur votre infrastructure. Commencez par <a href="${routes.createApp}">en créer une !</a>`,
		'app.new': 'Nouvelle application',
		'app.edit': "Modifier l'application",
		'app.delete': "Supprimer l'application",
		'app.delete.confirm': (name: string) => `Voulez-vous vraiment supprimer l'application ${name} ?

Cette action est IRRÉVERSIBLE et supprimera TOUTES LES DONNÉES associées sur chaque environnement : conteneurs, images, volumes, logs et networks.`,
		'app.delete.failed': 'Erreur de suppression',
		'app.no_deployments': 'Aucun déploiement',
		'app.name.help':
			"Le nom de l'application détermine le sous-domaine utilisé par les déploiements. C'est pourquoi vous devez <strong>uniquement</strong> utiliser des <strong>caractères alphanumériques</strong> et un <strong>nom unique</strong> au sein des différentes cibles.",
		'app.how': 'Comment les services sont-ils exposés par checkship ?',
		'app.how.placeholder.name': '<nom-app>',
		'app.how.placeholder.scheme': '<scheme cible>://',
		'app.how.placeholder.url': '<url cible>',
		'app.how.description':
			'Les services possédant des <strong>mappings de ports</strong> seront exposés selon ces conventions :',
		'app.how.env': 'Environnement',
		'app.how.default': 'Service par défaut (premier par ordre alphabétique)',
		'app.how.others': 'Autres services exposés (exemple: <code>dashboard</code>)',
		'app.how.others.title': 'Autres services exposés (exemple: dashboard)',
		'app.general': 'Paramètres généraux',
		'app.vcs': 'Contrôle de version',
		'app.vcs.enabled': 'Utiliser un contrôle de version ?',
		'app.vcs.help':
			"Si vous n'utilisez pas de contrôle de version, vous pourrez toujours déployer manuellement votre application.",
		'app.vcs.token': "Jeton d'accès",
		'app.vcs.token.help': `Jeton utilisé pour vous authentifier auprès du dépôt. Généralement connu sous le nom de <strong>Jeton d'accès personnel</strong>, vous pouvez trouver des instructions pour <a href="https://docs.github.com/authentication/keeping-your-account-and-data-secure/creating-a-personal-access-token" rel="noopener noreferrer" target="_blank">Github</a> et <a href="https://docs.gitlab.com/ee/user/profile/personal_access_tokens.html" rel="noopener noreferrer" target="_blank">Gitlab</a>, laissez vide si le dépôt est public.`,
		'app.environment.production': 'Paramètres de production',
		'app.environment.staging': 'Paramètres de staging',
		'app.environment.target': 'Cible de déploiement',
		'app.environment.target.changed': 'Cible mise à jour',
		'app.environment.target.changed.description': (name: string) =>
			`Si vous changez de cible, toutes les ressources liées à cette application déployées par checkship sur <strong>${name}</strong> seront <strong>SUPPRIMÉES</strong> et un déploiement sur la nouvelle cible sera programmé si possible. Si vous devez sauvegarder quelque chose, faites le avant de changer la cible.`,
		'app.environment.vars': "Variables d'environnement",
		'app.environment.vars.service.add': 'Ajouter un service',
		'app.environment.vars.service.delete': 'Supprimer le service',
		'app.environment.vars.service.name': 'Nom du service',
		'app.environment.vars.service.name.help': (latestServices?: string[]) =>
			"Nom du service tel qu'apparaissant dans votre fichier de service." +
			(latestServices?.length
				? ` Derniers services déployés : ${latestServices
						?.map((s) => `<code>${s}</code>`)
						.join(', ')}`
				: ''),
		'app.environment.vars.service.env': "Variables d'environnement",
		'app.environment.vars.blankslate': 'Aucune variable définie.',
		// Target
		'target.not_found':
			"Il semblerait que la cible que vous recherchez n'existe pas. Retournez à la",
		'target.not_found.cta': 'page des cibles',
		'target.new': 'Nouvelle cible',
		'target.delete': 'Supprimer la cible',
		'target.delete.failed': 'Erreur de suppression',
		'target.reconfigure': 'Reconfigurer',
		'target.reconfigure.failed': 'Erreur de reconfiguration',
		'target.failed': 'La configuration de la cible a échouée',
		'target.ready': 'La configuration de la cible a réussie',
		'target.ready.description': function (at: string) {
			return `L'infrastructure nécessaire demandée le ${this.datetime(
				at
			)} a été déployée. Si quelque chose se passe mal, vous pouvez utiliser le bouton reconfigurer pour essayer à nouveau.`;
		},
		'target.configuring': 'Configuration de la cible en cours',
		'target.configuring.description': `L'infrastructure nécessaire est en cours de déploiement, veuillez patienter.`,
		'target.blankslate': `Aucune cible pour le moment. <br />Les cibles déterminent sur quel hôte vos <strong>applications seront déployées</strong>. Commencez par <a href="${routes.createTarget}">en créer une !</a>`,
		'target.general': 'Paramètres généraux',
		'target.name.help': `Le nom est utilisé uniquement pour l'affichage. Vous pouvez choisir ce que vous voulez.`,
		'target.manual_proxy': 'Proxy manuel',
		'target.automatic_proxy_configuration': 'Exposer les services automatiquement',
		'target.automatic_proxy_configuration.help': `Si activé, un proxy sera déployé sur la cible et vos services seront <a target="_blank" href="https://checkship-deploy.github.io/checkship/reference/providers/docker.html#exposing-services">automatiquement exposés</a>. Si désactivé, vous <strong>devrez faire le nécessaire</strong> pour rendre vos services accessibles en utilisant la méthode de votre choix. Changer ce paramètre <strong>pourra</strong> nécessiter le redéploiement de vos applications.`,
		'target.url.help': `Toutes les applications déployées sur cette cible seront disponibles en tant que <strong>sous-domaine</strong> de cette URL racine (sans sous-chemin). Elle doit être <strong>unique</strong> parmi les cibles. Vous <strong>DEVEZ</strong> configurer un <strong>DNS wildcard</strong> pour les sous-domaines de telle sorte que <code>*.&lt;url configurée&gt;</code> redirige vers l'IP de cette cible.`,
		'target.provider': 'Fournisseur',
		'target.provider.docker.help': "Docker engine <strong>DOIT</strong> être installé sur l'hôte.",
		'target.docker.is_remote': 'Docker distant',
		'target.docker.is_remote.help':
			'Se connecter à un démon docker distant en SSH. <strong>Ne peut pas être changé</strong> après la création.',
		'target.docker.host': 'Hôte',
		'target.docker.host.help':
			'Hôte ou adresse IP de la machine distante. <strong>Ne peut pas être changé</strong> après la création.',
		'target.docker.user': 'Utilisateur',
		'target.docker.port': 'Port',
		'target.docker.private_key': 'Clé privée',
		'target.docker.private_key.help': `Clé privée utilisée pour se connecter en SSH. <strong>Assurez-vous qu'elle inclut un saut de ligne à la fin</strong> ou la connexion risque d'échouer avec une erreur <code>invalid format</code>. Vous <strong>DEVEZ</strong> ajouter la clé publique correspondante dans le fichier <code>~/.ssh/authorized_keys</code> du serveur. Phrases secrètes non supportées pour le moment.`,
		'target.delete.confirm': (name: string) => `Voulez-vous vraiment supprimer la cible ${name} ?
	
Cette action est IRRÉVERSIBLE et supprimera TOUTES LES DONNÉES sur cette cible: conteneurs, images, volumes et networks.`,
		'target.delete.confirm_failed_status': (name: string) =>
			`Voulez-vous vraiment supprimer la cible ${name} ?
		
Il semblerait que la cible ne soit plus joignable. Si vous décidez de la supprimer maintenant, les ressources ne seront pas nettoyées.

Vous devriez probablement essayer de rendre la cible accessible avant de la supprimer.`,
		'target.reconfigure.confirm': `L'infrastructure nécessaire sera redéployée sur la cible. En êtes-vous sûr ?`,
		// Jobs
		'jobs.status': 'Statut',
		'jobs.resource': 'Message / Ressource',
		'jobs.dates': 'Créée le / Pas avant',
		'jobs.error': 'erreur',
		'jobs.payload': 'charge utile',
		'jobs.policy': 'politique',
		'jobs.policy.preserve_group_order': "Préserve l'ordre au sein du groupe en cas d'erreur",
		'jobs.policy.wait_others_resource_id': "Attend l'achèvement des tâches sur cette ressource",
		'jobs.policy.cancellable': 'Annulable',
		'jobs.policy.mergeable': 'Fusionnable',
		'jobs.group': 'groupe',
		'jobs.cancel': 'Annuler la tâche',
		'jobs.cancel.confirm': 'Voulez-vous vraiment annuler la tâche ?',
		// Jobs names
		'deployment.command.cleanup_app': "Nettoyage de l'application",
		'deployment.command.cleanup_target': 'Nettoyage de la cible',
		'deployment.command.delete_app': "Suppression de l'application",
		'deployment.command.delete_target': 'Suppression de la cible',
		'deployment.command.configure_target': 'Configuration de la cible',
		'deployment.command.deploy': "Déploiement de l'application",
		// Registries
		'registry.new': 'Nouveau registre',
		'registry.blankslate': `Aucun registre pour le moment. <br />Si certaines de vos images sont hébergées sur <strong>des registres privés</strong>, <a href="${routes.createRegistry}">configurer les ici</a> de manière à les rendre disponibles.`,
		'registry.not_found':
			"Il semblerait que le registre que vous recherchez n'existe pas. Retournez à la",
		'registry.not_found.cta': 'page des registres',
		'registry.delete': 'Supprimer le registre',
		'registry.delete.confirm': (name: string) =>
			`Voulez-vous vraiment supprimer le registre ${name} ?`,
		'registry.delete.failed': 'Erreur de suppression',
		'registry.general': 'Paramètres généraux',
		'registry.url.help':
			"Url du registre. Pour un registre privé Docker Hub, utiliser l'url <code>https://index.docker.io/v1/</code>.",
		'registry.authentication': 'Authentification',
		'registry.auth': 'Authentification nécessaire',
		'registry.auth.help': 'Le registre nécessite t-il une authentification ?',
		'registry.username': "Nom d'utilisateur",
		'registry.password': 'Mot de passe',
		'registry.name.help': `Le nom est utilisé uniquement pour l'affichage. Vous pouvez choisir ce que vous voulez.`,
		// Account
		'profile.my': 'mon profil',
		'profile.logout': 'se déconnecter',
		'profile.password': 'Nouveau mot de passe',
		'profile.password.help': 'Laissez vide si vous ne souhaitez pas changer de mot de passe.',
		'profile.informations': 'Informations du compte',
		'profile.interface': 'Interface utilisateur',
		'profile.locale': 'Langue',
		'profile.integration': 'Intégration',
		'profile.integration.title': 'Intégration Continue',
		'profile.integration.description':
			"Si vous souhaitez déclencher un déploiement pour une application, vous aurez besoin de ce jeton. Vous pouvez également <strong>Afficher la commande cURL</strong> depuis la page de déploiement et l'utiliser dans votre pipeline car il inclut le jeton dans l'en-tête approprié.",
		'profile.key': 'Clé API',
		'profile.key.help':
			"Passez ce jeton dans l'entête <code>Authorization: Bearer</code> pour communiquer avec l'API checkship. <strong>Vous devez le garder secret !</strong>",
		'profile.key.refresh': "Regénérer la clé d'API",
		'profile.key.refresh.confirm':
			"L'ancienne clé ne sera plus valide. Confirmez-vous cette action ?",
		// Deployment
		'deployment.new': 'Nouveau déploiement',
		'deployment.deploy': 'Déployer',
		'deployment.redeploy': 'Redéployer',
		'deployment.redeploy.confirm': (number: number) =>
			`Le déploiement #${number} sera redéployé. La dernière version des variables d'environnement sera utilisée. Confirmez-vous cette action ?`,
		'deployment.redeploy.failed': 'Erreur lors du redéploiement',
		'deployment.promote': 'Promouvoir',
		'deployment.promote.confirm': (number: number) =>
			`Le déploiement #${number} sera promu sur l'environnement de production. Confirmez-vous cette action ?`,
		'deployment.promote.failed': 'Erreur lors de la promotion',
		'deployment.blankslate': (app: string) =>
			`Aucun déploiement trouvé. Commencez par <a href="${routes.createDeployment(
				app
			)}">en créer un !</a>`,
		'deployment.environment': 'Environnement',
		'deployment.payload': 'Charge utile',
		'deployment.payload.toggle_curl_command': 'Afficher / Masquer la commande cURL',
		'deployment.payload.kind': 'Type',
		'deployment.payload.raw': 'fichier compose',
		'deployment.payload.raw.content': 'Contenu',
		'deployment.payload.raw.content.help':
			'Contenu du fichier de services (compose.yml si vous utilisez Docker Compose par exemple).',
		'deployment.payload.archive': 'archive (tar.gz)',
		'deployment.payload.vcs': 'git',
		'deployment.payload.vcs.branch': 'Branche',
		'deployment.payload.vcs.commit': 'Commit',
		'deployment.payload.vcs.commit.help':
			'Commit spécifique à déployer. Laissez vide pour déployer le dernier commit de la branche.',
		'deployment.logs': 'Logs de déploiement',
		'deployment.outdated': 'Déploiement obsolète',
		'deployment.outdated.description': (href: string) =>
			`Vous visualisez un ancien déploiement et les URLs exposées ici représentent ce qui a été exposé au moment du déploiement. Accéder à <a href="${href}">la dernière version</a>.`,
		'deployment.started_at': 'démarré à',
		'deployment.finished_at': 'terminé à',
		'deployment.queued_at': 'demandé à',
		'deployment.duration': 'durée',
		'deployment.services': 'services déployés',
		'deployment.branch': 'branche',
		'deployment.commit': 'commit',
		'deployment.error_code': 'code erreur',
		'deployment.details_tooltip': (number: number) =>
			`Voir les détails et logs du déploiement #${number}`,
		'deployment.not_found':
			"Il semblerait que le déploiement que vous recherchez n'existe pas. Retournez à",
		'deployment.not_found.cta': "la page d'application",
		'deployment.waiting_for_logs': 'En attente des logs...',
		'deployment.target': 'deployé sur',
		'deployment.target.deleted': 'cible supprimée',
		// Menu
		'menu.toggle': 'Basculer le menu',
		// Breadcrumb
		'breadcrumb.home': 'Accueil',
		'breadcrumb.applications': 'Applications',
		'breadcrumb.settings': 'Paramètres',
		'breadcrumb.application.new': 'Nouvelle application',
		'breadcrumb.application.settings': (name: string) => `Paramètres de ${name}`,
		'breadcrumb.deployment.new': 'Nouveau déploiement',
		'breadcrumb.deployment.name': (number: number) => `Déploiement #${number}`,
		'breadcrumb.targets': 'Cibles',
		'breadcrumb.target.new': 'Nouvelle cible',
		'breadcrumb.target.settings': (name: string) => `Paramètres de ${name}`,
		'breadcrumb.jobs': 'Tâches',
		'breadcrumb.registries': 'Registres',
		'breadcrumb.registry.new': 'Nouveau registre',
		'breadcrumb.registry.settings': (name: string) => `Paramètes de ${name}`,
		'breadcrumb.profile': 'Profil',
		'breadcrumb.not_found': 'Ressource introuvable',
		// Footer
		'footer.description': (version: string) =>
			`checkship v${version.substring(0, 16)} — Plateforme de déploiement auto-hébergée`,
		'footer.source': 'Source',
		'footer.documentation': 'Docs',
		'footer.credits': 'Crédits',
		// Shared
		'console.copy_to_clipboard': 'Copier dans le presse-papiers',
		'console.select_all': 'Tout sélectionner',
		'panel.hint': 'Afficher / Masquer',
		'datatable.no_data': 'Aucune donnée à afficher',
		'datatable.toggle': 'Afficher / masquer les détails',
		cleanup_requested: 'Suppression demandée',
		'cleanup_requested.description': function (date: DateValue) {
			return `La suppression a été demandée le ${this.date(date)} et sera traitée sous peu.`;
		},
		page_n_of_m: (n: number, m: number) => `Page ${n} de ${m}`,
		file: 'Fichier',
		command: 'Commande',
		previous: 'Précédent',
		next: 'Suivant',
		and: 'et',
		save: 'Enregistrer',
		create: 'Créer',
		error: 'Erreur',
		name: 'Nom',
		email: 'Email',
		url: 'Url',
		password: 'Mot de passe',
		required: 'Requis',
		invalid_email: 'Email invalide',
		unexpected_error: 'Une erreur imprévue est survenue.',
		'unexpected_error.description':
			"<p>Quelque chose s'est mal passé. Tentez de rafraichir la page.</p><p>Si le problème persiste, veuillez contacter l'administrateur.</p>",
		app_name_already_taken: "Nom d'application déjà utilisé sur cette cible",
		git_branch_not_found: 'Branche non trouvée',
		git_remote_not_reachable: 'Origine injoignable',
		invalid_email_or_password: 'Email ou mot de passe invalide',
		invalid_app_name: "Nom d'application invalide",
		url_already_taken: 'Url déjà utilisée',
		config_already_taken: 'Une cible pour cet hôte existe déjà',
		invalid_host: 'Hôte invalide',
		invalid_ssh_key: 'Clé SSH invalide',
		target_in_use:
			"La cible est en cours d'utilisation par au moins une application et ne peut pas être supprimée.",
		// Shell (DESIGN.md §5)
		'nav.main': 'Principal',
		'nav.workspace': 'Espace de travail',
		'nav.recent': 'Récents',
		'nav.documentation': 'Documentation',
		'nav.documentation.label': 'Documentation, ouvre un nouvel onglet',
		'nav.theme': 'Thème',
		'nav.collapse': 'Réduire la barre latérale',
		'nav.expand': 'Déplier la barre latérale',
		'nav.open_menu': 'Ouvrir le menu',
		'nav.close_menu': 'Fermer le menu',
		'nav.targets_failing': (count: number) =>
			count === 1 ? '1 cible en échec' : `${count} cibles en échec`,
		'nav.jobs_queued': (count: number) =>
			count === 1 ? '1 tâche en attente' : `${count} tâches en attente`,
		'nav.version': (version: string) => `checkship ${version}`,
		'theme.system': 'Système',
		'theme.light': 'Clair',
		'theme.dark': 'Sombre',
		'theme.switch': (current: string) => `Thème : ${current}. Changer de thème`,
		'account.menu': (email: string) => `Compte : ${email}`,
		'account.signed_in': 'Connecté',
		'account.profile': 'Profil',
		'account.api_key': "Clé d'API pour la CI",
		'account.sign_out': 'Se déconnecter',
		'apps.summary': (count: number, failed: number, queued: number) =>
			[
				count === 1 ? '1 application' : `${count} applications`,
				failed > 0
					? failed === 1
						? '1 déploiement de production en échec'
						: `${failed} déploiements de production en échec`
					: undefined,
				queued > 0
					? queued === 1
						? '1 tâche en attente'
						: `${queued} tâches en attente`
					: undefined
			]
				.filter(Boolean)
				.join(' · '),
		'apps.column.application': 'Application',
		'apps.column.production': 'Production',
		'apps.column.staging': 'Staging',
		'apps.no_deployment': 'Aucun déploiement',
		'apps.on_target': (name: string) => `sur la cible ${name}`,
		'apps.deletion_requested': 'Suppression demandée',
		'apps.deletion_requested.description':
			'Ses ressources sont en cours de suppression sur les cibles.',
		'apps.empty.title': "Aucune application pour l'instant",
		'apps.empty.description':
			"Une application regroupe les services d'un fichier compose, déployés en production et en staging.",
		'targets.summary': (count: number, failing: number) =>
			[count === 1 ? '1 cible' : `${count} cibles`, failing > 0 ? `${failing} en échec` : undefined]
				.filter(Boolean)
				.join(' · '),
		'targets.column.target': 'Cible',
		'targets.column.provider': 'Fournisseur',
		'targets.column.state': 'État',
		'targets.local': 'Moteur Docker local',
		'targets.remote': (host: string) => `Docker via SSH · ${host}`,
		'targets.manual_proxy': 'Services non exposés automatiquement',
		'targets.empty.title': "Aucune cible pour l'instant",
		'targets.empty.description':
			'Une cible est un hôte Docker sur lequel checkship déploie, sur cette machine ou via SSH.',
		'registries.summary': (count: number) => (count === 1 ? '1 registre' : `${count} registres`),
		'registries.column.registry': 'Registre',
		'registries.column.auth': 'Authentification',
		'registries.auth.none': 'Sans authentification',
		'registries.auth.user': (username: string) => `Connecté en tant que ${username}`,
		'registries.empty.title': 'Aucun registre personnalisé',
		'registries.empty.description':
			'Ajoutez-en un quand vos images sont dans un registre privé qui demande des identifiants.',
		'status.pending': 'En attente',
		'status.running': 'En cours',
		'status.failed': 'Échec',
		'status.succeeded': 'Réussi',
		'status.configuring': 'Configuration',
		'status.ready': 'Prête',
		'deployment.promote.title': 'Promouvoir en production',
		'deployment.promote.description': (number: number) =>
			`Le déploiement #${number} sera redéployé en production, avec la même source.`,
		'deployment.redeploy.description': (number: number, environment: string) =>
			`Le déploiement #${number} sera redéployé en ${environment}, avec la même source.`,
		'deployment.status_line': (environment: string, target: string) =>
			`${environment} · cible ${target}`,
		'deployment.finished_line': (time: string, duration: string) =>
			`terminé à ${time} · durée ${duration}`,
		'deployment.started_line': (time: string) => `démarré à ${time}`,
		'deployment.queued_line': (time: string) => `en file depuis ${time}`,
		'deployment.source': 'Source',
		'deployment.source.kind': 'Type',
		'deployment.source.git': 'Git',
		'deployment.source.raw': 'Fichier compose',
		'deployment.source.archive': 'Archive',
		'deployment.queued_by': 'Demandé par',
		'deployment.services.title': 'Services',
		'deployment.not_exposed': 'Non exposé',
		'deployment.logs.title': 'Journaux',
		'deployment.outdated.link': (number: number) => `Aller au déploiement #${number}`,
		'target.delete.description': (name: string) =>
			`Les ressources de ses applications sont supprimées de l'hôte. Saisissez ${name} pour confirmer.`,
		'target.delete.type_name': 'Nom de la cible',
		'target.reconfigure.title': 'Reconfigurer la cible',
		'target.create': 'Créer la cible',
		'target.remote.label': 'Hôte distant via SSH',
		'target.remote.description':
			'Désactivé : le moteur Docker de la machine qui exécute checkship.',
		'target.page.description':
			'Une cible est un hôte Docker sur lequel checkship déploie, sur cette machine ou via SSH.',
		'profile.signed_in_since': (date: string) => `Inscrit depuis le ${date}`,
		'profile.account': 'Compte',
		'profile.locale.help': 'La page se recharge dans la nouvelle langue.',
		'profile.content_width': 'Largeur du contenu',
		'profile.content_width.help':
			'Largeur maximale des pages sur les grands écrans : 840, 1020 ou 1160px. Le thème est en bas de la barre latérale.',
		'profile.content_width.compact': 'Compacte',
		'profile.content_width.default': 'Normale',
		'profile.content_width.wide': 'Large',
		'profile.saved': 'Profil enregistré',
		'profile.key.refresh.title': "Régénérer la clé d'API ?",
		'profile.key.regenerated': "Clé d'API régénérée",
		copy: 'Copier',
		copied: 'Copié dans le presse-papiers',
		cancel: 'Annuler',
		confirm: 'Confirmer',
		delete: 'Supprimer'
	}
} as const satisfies Locale<AppTranslations>;
