import { Link } from "../router";

const effective = "13 November 2026";

function Shell({ title, updated, children }: { title: string; updated?: string; children: React.ReactNode }) {
  return (
    <article style={{ maxWidth: 720 }}>
      <Link to="/" className="small muted">
        ← Back to Crew
      </Link>

      <h1 className="page-title mt-16" style={{ marginBottom: 6 }}>
        {title}
      </h1>
      <p className="small muted mb-24">Last updated {updated ?? effective}</p>

      <div className="stack" style={{ color: "var(--ink-soft)", lineHeight: 1.7 }}>
        {children}
      </div>

      <hr className="divider" />
      <p className="tiny muted">
        Questions? Email <a href="mailto:support@paytrackify.com">support@paytrackify.com</a>.
      </p>
    </article>
  );
}

export function Privacy() {
  return (
    <Shell title="Privacy Policy">
      <p>
        Crew is a small tool for giving AI helpers work. This page explains, in plain words, what we keep and why.
      </p>

      <h2 className="section-title">What we store</h2>
      <ul>
        <li>Your name and email address, so we know who you are.</li>
        <li>The tasks you create, the helpers you add, the notes you write, and the activity that follows.</li>
        <li>Basic technical details when you sign in, such as the device name and IP address, so you can sign out devices you don't recognise.</li>
        <li>Company details you enter, such as your website, support email and what the business does.</li>
        <li>AI provider keys you choose to add. They are encrypted on the server and never shown again in the app.</li>
      </ul>

      <h2 className="section-title">What we don't do</h2>
      <ul>
        <li>We don't sell your data.</li>
        <li>We don't read your tasks to train models.</li>
        <li>We don't ask for a password, because there isn't one.</li>
      </ul>

      <h2 className="section-title">Who else is involved</h2>
      <p>
        Sign-in is handled by PayTrackify, which sends your one-time code. When you connect an AI provider, your
        business details and task text are sent to that provider so your helpers can do the work. Their terms and
        privacy policy also apply. We name every provider and link to its key page inside the app.
      </p>

      <h2 className="section-title">Your choices</h2>
      <p>
        You can download everything we hold from <Link to="/settings">Settings → Privacy & data</Link>, and delete your
        workspace from <Link to="/settings">Settings → Danger zone</Link>. Deleting removes your tasks, helpers, notes,
        and activity. Your PayTrackify account is separate and stays as it is.
      </p>

      <h2 className="section-title">Keeping it safe</h2>
      <p>
        Sign-in codes expire quickly, sessions can be revoked at any time, and access is limited to your own
        workspace. If you think someone else has access, sign out all devices and contact us.
      </p>
    </Shell>
  );
}

export function Terms() {
  return (
    <Shell title="Terms of Agreement">
      <p>By using Crew you agree to these simple rules. If you don't agree, please don't use Crew.</p>

      <h2 className="section-title">Your account</h2>
      <p>
        You sign in with a one-time code sent to your email. Keep access to that inbox secure. You are responsible
        for the work you ask helpers to do and for the people you invite.
      </p>

      <h2 className="section-title">AI providers</h2>
      <p>
        Crew does not include a model. You connect your own provider account and are responsible for its usage,
        costs and terms. Keys are encrypted at rest, used only to run your helpers, and can be removed at any time.
      </p>

      <h2 className="section-title">Acceptable use</h2>
      <ul>
        <li>Don't use Crew to break the law, harm people, or infringe someone else's rights.</li>
        <li>Don't try to access another person's workspace or overload the service.</li>
        <li>Don't use helpers to send spam or to impersonate people.</li>
      </ul>

      <h2 className="section-title">Your content</h2>
      <p>You own what you put into Crew. We store it so the product can work, and nothing more.</p>

      <h2 className="section-title">Availability</h2>
      <p>
        Crew is provided as-is. We work hard to keep it running, but we can't promise it will never be unavailable.
        Keep your own copy of anything you can't afford to lose.
      </p>

      <h2 className="section-title">Ending use</h2>
      <p>
        You can stop using Crew at any time and delete your workspace in Settings. We may suspend accounts that
        abuse the service.
      </p>
    </Shell>
  );
}

export function Support() {
  return (
    <Shell title="Support">
      <p>Stuck? Most things are fixed in a minute. Start here.</p>

      <h2 className="section-title">I didn't get my code</h2>
      <p>
        Check your spam or junk folder first. Codes can take up to a minute. If it still hasn't arrived, wait for
        the timer and choose “Send a new code”. Make sure the email address has no typos.
      </p>

      <h2 className="section-title">My code says it's wrong</h2>
      <p>
        Codes expire after 10 minutes and work once. Ask for a new one and type the freshest code. If you've tried
        several times, wait a few minutes before trying again.
      </p>

      <h2 className="section-title">My helper isn't picking up tasks</h2>
      <p>
        Open the helper, make a connection key, and run the command on the computer that should do the work. Keep
        that window open. Then add a task and assign it to that helper.
      </p>

      <h2 className="section-title">I want my data or my account gone</h2>
      <p>
        Go to <Link to="/settings">Settings</Link>. You can download everything under “Privacy & data”, and delete your
        workspace under “Danger zone”.
      </p>

      <h2 className="section-title">Still stuck?</h2>
      <p>
        Email <a href="mailto:support@paytrackify.com">support@paytrackify.com</a> with what you were trying to do and
        what happened. We read every message.
      </p>
    </Shell>
  );
}

export function NotFound() {
  return (
    <div className="empty">
      <div className="empty-title">We couldn't find that page</div>
      <p className="small">The link may be old, or the page moved.</p>
      <Link to="/" className="btn mt-16" style={{ display: "inline-flex" }}>
        Go home
      </Link>
    </div>
  );
}
