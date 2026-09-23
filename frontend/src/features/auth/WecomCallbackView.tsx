import { Loader2 } from 'lucide-react';

// WecomCallbackView 企业微信整页回跳处理视图：展示回调处理进度或失败提示，样式与登录页一致
export function WecomCallbackView({ error }: { error: string }) {
  return (
    <main
      data-cmp="Login"
      className="relative flex min-h-dvh items-center justify-center overflow-hidden px-4 py-8 sm:px-6"
      style={{
        background:
          'radial-gradient(circle at 50% 0%, rgba(59,130,246,0.24), transparent 30%), radial-gradient(circle at 12% 22%, rgba(6,182,212,0.18), transparent 28%), radial-gradient(circle at 88% 82%, rgba(16,185,129,0.14), transparent 30%), var(--zl-login-bg)',
        color: 'var(--zl-text)',
      }}
    >
      <div className="zl-login-grid absolute inset-0" aria-hidden="true" />
      <div
        className="zl-login-orb absolute left-[-6rem] top-20 h-64 w-64 rounded-full"
        aria-hidden="true"
      />
      <div
        className="zl-login-orb absolute bottom-[-4rem] right-[-5rem] h-80 w-80 rounded-full"
        aria-hidden="true"
      />
      <section
        className="relative z-10 flex w-full max-w-[420px] flex-col items-center gap-4 rounded-[24px] p-8 text-center"
        style={{
          background: 'var(--zl-login-panel-bg)',
          border: '1px solid var(--zl-border)',
          backdropFilter: 'blur(18px)',
          boxShadow: 'var(--zl-login-panel-shadow)',
        }}
      >
        {error ? (
          <>
            <p className="text-sm leading-6" style={{ color: '#fca5a5' }} role="alert">
              {error}
            </p>
            <button
              type="button"
              onClick={() => window.location.replace('/login')}
              className="zl-action-button rounded-xl px-4 py-2 text-sm font-medium"
              style={{
                borderColor: 'var(--zl-border)',
                background: 'var(--zl-control-bg)',
                color: 'var(--zl-accent-text)',
              }}
            >
              返回登录
            </button>
          </>
        ) : (
          <>
            <Loader2 size={28} className="zl-spinner" />
            <p className="text-sm font-medium" style={{ color: 'var(--zl-text)' }}>
              正在处理企业微信授权，请稍候…
            </p>
          </>
        )}
      </section>
    </main>
  );
}
