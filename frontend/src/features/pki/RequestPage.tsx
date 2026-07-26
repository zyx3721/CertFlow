import { useEffect, useMemo, useRef, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { ArrowRight, CheckCircle2, FileKey2, Upload } from 'lucide-react';
import { toast } from 'sonner';
import { Field } from '@/components/field';
import { PageHeader } from '@/components/page-header';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import {
  createCertificateRequest,
  getCAs,
  inspectCertificateCSR,
  previewCertificateCSR,
  type CertificatePurpose,
  type KeyAlgorithm,
} from '@/lib/pki';
import { pkiQueryKeys, usePageRefresh } from './support';
import { RequestPreview } from './RequestPreview';
import {
  customValidityOption,
  requestedValidityDays as calculateValidityDays,
  RequestValidityField,
  type CustomValidityUnit,
  validityLabel as getValidityLabel,
} from './RequestValidityField';

const requestKeys = [pkiQueryKeys.cas] as const;
const initialSubject = {
  cn: '',
  o: 'Acme Corp',
  ou: 'Security',
  c: 'CN',
  st: 'Shanghai',
  l: 'Shanghai',
};
const ipv4LikePattern = /^\d+(?:\.\d+){3}$/;

type Step = 1 | 2 | 3;
type CsrInputMode = 'generate' | 'manual';
type SubjectForm = typeof initialSubject;

function isValidIPv4(value: string) {
	if (!ipv4LikePattern.test(value)) return false;
	return value.split('.').every(part => Number(part) >= 0 && Number(part) <= 255);
}

function looksLikeIPAddress(value: string) {
	return /^[0-9.]+$/.test(value);
}

export function RequestPage() {
  usePageRefresh(requestKeys);
  const queryClient = useQueryClient();
  const caQuery = useQuery({ queryKey: pkiQueryKeys.cas, queryFn: getCAs });
  const issuingCAs = useMemo(
    () =>
      (caQuery.data?.items ?? []).filter(
        item => item.type === 'issuing' && item.status === 'active'
      ),
    [caQuery.data?.items]
  );
  const [step, setStep] = useState<Step>(1);
  const [caID, setCAID] = useState('');
  const [subject, setSubject] = useState<SubjectForm>(initialSubject);
  const [algorithm, setAlgorithm] = useState<KeyAlgorithm>('RSA-2048');
  const [purpose, setPurpose] = useState<CertificatePurpose>('server');
  const [validityOption, setValidityOption] = useState('365');
  const [customValidityValue, setCustomValidityValue] = useState('');
  const [customValidityUnit, setCustomValidityUnit] = useState<CustomValidityUnit>('day');
  const [sanText, setSanText] = useState('');
  const [csrInputMode, setCSRInputMode] = useState<CsrInputMode>('generate');
  const [csrPEM, setCSRPEM] = useState('');
  const [csrPreview, setCSRPreview] = useState<{ csrPEM: string; privateKeyPEM: string } | null>(
    null
  );
  const [isGeneratingCSR, setIsGeneratingCSR] = useState(false);
  const manualCSRParseSequence = useRef(0);

  useEffect(() => {
    if (issuingCAs.length === 0) return;
    setCAID(current => (issuingCAs.some(item => item.id === current) ? current : issuingCAs[0].id));
  }, [issuingCAs]);

  const sanValues = useMemo(
    () =>
      sanText
        .split(',')
        .map(value => value.trim())
        .filter(Boolean),
    [sanText]
  );
  const subjectText = useMemo(
    () =>
      [
        ['CN', subject.cn],
        ['O', subject.o],
        ['OU', subject.ou],
        ['C', subject.c],
        ['ST', subject.st],
        ['L', subject.l],
      ]
        .filter(([, value]) => value.trim())
        .map(([key, value]) => `${key}=${value.trim()}`)
        .join(', '),
    [subject]
  );
  const selectedCA = useMemo(
    () => issuingCAs.find(item => item.id === caID),
    [caID, issuingCAs]
  );
  const requestedValidityDays = useMemo(() => {
    return calculateValidityDays(validityOption, customValidityValue, customValidityUnit);
  }, [customValidityUnit, customValidityValue, validityOption]);
  const validityLabel = useMemo(() => {
    return getValidityLabel(validityOption, customValidityValue, customValidityUnit);
  }, [customValidityUnit, customValidityValue, validityOption]);
  const validationError = useMemo(() => {
    const normalizedCSR = csrPEM.trim();

    // Required fields are evaluated before all format constraints.
    if (!caID) return '请选择签发 CA';
    if (!subject.cn.trim()) return '通用名称（CN）不能为空';
    if (!sanText.trim()) return '证书域名不能为空';
    if (csrInputMode === 'manual' && !normalizedCSR) return '请上传或填写 CSR 内容';
    if (validityOption === customValidityOption && !customValidityValue.trim()) {
      return '自定义有效期不能为空';
    }

    if (!selectedCA) return '请选择有效的签发 CA';
    if (validityOption === customValidityOption && !/^\d+$/.test(customValidityValue.trim())) {
      return '自定义有效期必须为正整数';
    }
    if (!Number.isInteger(requestedValidityDays) || requestedValidityDays < 1 || requestedValidityDays > 7300) {
      return '证书有效期必须在 1 到 7300 天之间';
    }
	if (sanValues.length > 100) return 'SAN 数量不能超过 100';
    if (subject.c.trim().length !== 2) return '国家代码需为 2 位，例如 CN';
    if (sanText.includes('，') || sanValues.some(value => /\s/.test(value))) {
      return '多个证书域名请使用英文逗号分隔';
    }
	if (sanValues.some(value => looksLikeIPAddress(value) && !isValidIPv4(value))) {
      return 'IP 地址格式不合法';
    }
    if (
      csrInputMode === 'manual' &&
      !normalizedCSR.includes('-----BEGIN CERTIFICATE REQUEST-----')
    ) {
      return 'CSR 内容格式不正确';
    }
    const requestedNotAfter = new Date();
    requestedNotAfter.setDate(requestedNotAfter.getDate() + requestedValidityDays);
    if (requestedNotAfter > new Date(selectedCA.notAfter)) {
      return '证书有效期不能超过签发 CA 有效期';
    }
    return '';
  }, [
    caID,
    csrInputMode,
    csrPEM,
    customValidityValue,
    requestedValidityDays,
    sanText,
    sanValues,
    selectedCA,
    subject.c,
    subject.cn,
    validityOption,
  ]);

  const mutation = useMutation({
    mutationFn: createCertificateRequest,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: pkiQueryKeys.certificates });
      void queryClient.invalidateQueries({ queryKey: pkiQueryKeys.workflows });
      void queryClient.invalidateQueries({ queryKey: pkiQueryKeys.audits });
      setStep(3);
      toast.success(`证书申请已提交，CN: ${subject.cn.trim()}`);
    },
    onError: error => toast.error(error instanceof Error ? error.message : '证书申请提交失败'),
  });

  function updateSubject(key: keyof SubjectForm, value: string) {
    setSubject(current => ({ ...current, [key]: value }));
  }

  async function goNext() {
    if (validationError) {
      toast.error(validationError);
      return;
    }
    setIsGeneratingCSR(true);
    try {
      if (csrInputMode === 'manual') {
		await inspectCertificateCSR(csrPEM.trim());
        setCSRPreview({ csrPEM: csrPEM.trim(), privateKeyPEM: '' });
      } else {
        setCSRPreview(
          await previewCertificateCSR({
            subject: subjectText,
            algorithm,
            san: sanValues,
          })
        );
      }
      setStep(2);
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'CSR 生成失败');
    } finally {
      setIsGeneratingCSR(false);
    }
  }

  function submit() {
    if (validationError) {
      setStep(1);
      toast.error(validationError);
      return;
    }
    mutation.mutate({
      caID,
      commonName: subject.cn.trim(),
      subject: subjectText,
      algorithm,
      purpose,
      san: sanValues,
      validityDays: requestedValidityDays,
      csrPEM: csrPreview?.csrPEM,
      privateKeyPEM: csrInputMode === 'generate' ? csrPreview?.privateKeyPEM : undefined,
    });
  }

  function reset() {
    setStep(1);
    setSubject(initialSubject);
    setAlgorithm('RSA-2048');
    setPurpose('server');
    setValidityOption('365');
    setCustomValidityValue('');
    setCustomValidityUnit('day');
    setSanText('');
    setCSRInputMode('generate');
    setCSRPEM('');
    setCSRPreview(null);
  }

  async function updateManualCSR(value: string) {
    setCSRPEM(value);
    setCSRPreview(null);
    const normalizedCSR = value.trim();
    if (!normalizedCSR.includes('-----BEGIN CERTIFICATE REQUEST-----')) return;

    const sequence = ++manualCSRParseSequence.current;
    try {
      const parsed = await inspectCertificateCSR(normalizedCSR);
      if (sequence !== manualCSRParseSequence.current) return;
      setSubject(current => ({
        cn: parsed.subject.commonName || current.cn,
        o: parsed.subject.org || current.o,
        ou: parsed.subject.orgUnit || current.ou,
        c: parsed.subject.country || current.c,
        st: parsed.subject.province || current.st,
        l: parsed.subject.locality || current.l,
      }));
      if (parsed.san.length > 0) {
        setSanText(parsed.san.join(', '));
      }
      setAlgorithm(parsed.algorithm);
      toast.success('CSR 内容已解析并回填');
    } catch (error) {
      if (sequence !== manualCSRParseSequence.current) return;
      toast.error(error instanceof Error ? error.message : 'CSR 解析失败');
    }
  }

  async function readCSRFile(file?: File) {
    if (!file) return;
    try {
      await updateManualCSR(await file.text());
    } catch {
      toast.error('CSR 文件读取失败');
    }
  }

  return (
    <div className="certflow-page-fill">
      <PageHeader title="证书申请" description="填写证书请求信息，预览确认后提交审批流程" />
      <form
        onSubmit={event => {
          event.preventDefault();
          if (step === 2) submit();
        }}
        className="flex min-h-0 flex-1 flex-col gap-4"
      >
        <div className="shrink-0">
          <RequestSteps step={step} />
        </div>

        {step === 1 ? (
          <section className="zl-surface-3d certflow-request-form-card certflow-page-card certflow-scroll-area rounded-xl border p-5">
            <div className="mb-5 flex flex-wrap items-start justify-between gap-3 border-b border-[var(--zl-border)] pb-4">
              <div className="flex items-center gap-3">
                <span className="grid h-9 w-9 place-items-center rounded-lg bg-blue-500/10 text-blue-500">
                  <FileKey2 size={17} />
                </span>
                <div>
                  <h2 className="text-sm font-semibold">证书主体信息</h2>
                  <p className="mt-1 text-xs text-[var(--zl-text-muted)]">
                    填写证书主题、域名、算法与签发 CA
                  </p>
                </div>
              </div>
              <span className="rounded-full border border-blue-500/20 bg-blue-500/10 px-2.5 py-1 text-xs font-medium text-blue-600 dark:text-blue-400">
                Step 1
              </span>
            </div>

            <div className="grid gap-4 sm:grid-cols-2">
              <Field label="通用名称（CN）" required>
                <Input
                  value={subject.cn}
                  onChange={event => updateSubject('cn', event.target.value)}
                  placeholder="例如：api.acme.com"
                />
              </Field>
              <Field label="组织（O）">
                <Input
                  value={subject.o}
                  onChange={event => updateSubject('o', event.target.value)}
                />
              </Field>
              <Field label="组织单位（OU）">
                <Input
                  value={subject.ou}
                  onChange={event => updateSubject('ou', event.target.value)}
                />
              </Field>
              <Field label="国家（C）">
                <Input
                  value={subject.c}
                  maxLength={2}
                  onChange={event => updateSubject('c', event.target.value.toUpperCase())}
                />
              </Field>
              <Field label="省份（ST）">
                <Input
                  value={subject.st}
                  onChange={event => updateSubject('st', event.target.value)}
                />
              </Field>
              <Field label="城市（L）">
                <Input
                  value={subject.l}
                  onChange={event => updateSubject('l', event.target.value)}
                />
              </Field>
            </div>

            <div className="mt-4">
              <Field label="证书域名" required hint="多个 DNS 名称或 IP 地址请用英文逗号分隔">
                <Input
                  value={sanText}
                  onChange={event => setSanText(event.target.value)}
                  placeholder="例如：api.acme.com, api-v2.acme.com"
                />
              </Field>
            </div>

            <div className="mt-4 grid gap-4 sm:grid-cols-2">
              <Field label="密钥算法">
                <Select
                  value={algorithm}
                  onValueChange={(value: KeyAlgorithm) => setAlgorithm(value)}
                >
                  <SelectTrigger className="font-normal">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {['RSA-2048', 'RSA-4096', 'ECDSA-P256', 'ECDSA-P384', 'ED25519'].map(value => (
                      <SelectItem key={value} value={value} className="font-normal">
                        {value}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </Field>
              <Field label="有效期">
                <RequestValidityField
                  option={validityOption}
                  customValue={customValidityValue}
                  customUnit={customValidityUnit}
                  onOptionChange={setValidityOption}
                  onCustomValueChange={setCustomValidityValue}
                  onCustomUnitChange={setCustomValidityUnit}
                />
              </Field>
            </div>

            <section className="mt-4 rounded-xl border border-[var(--zl-border)] bg-[var(--zl-control-bg-soft)]/50 p-4">
              <div className="flex flex-wrap items-center justify-between gap-3">
                <div>
                  <h3 className="text-sm font-semibold">CSR 来源</h3>
                  <p className="mt-1 text-xs text-[var(--zl-text-muted)]">
                    可由系统自动生成，也可上传或粘贴已有 CSR
                  </p>
                </div>
                <div className="inline-flex rounded-lg border border-[var(--zl-border)] bg-[var(--zl-control-bg-soft)]/70 p-1 text-xs shadow-inner shadow-black/5">
                  <button
                    type="button"
                    onClick={() => setCSRInputMode('generate')}
                    className={`rounded-md px-3 py-1.5 transition-colors ${
                      csrInputMode === 'generate'
                        ? 'border border-blue-500/30 bg-blue-500/10 text-blue-700 shadow-sm dark:text-blue-300'
                        : 'border border-transparent text-[var(--zl-text-muted)] hover:bg-[var(--zl-control-bg)] hover:text-[var(--zl-text)]'
                    }`}
                  >
                    系统生成
                  </button>
                  <button
                    type="button"
                    onClick={() => setCSRInputMode('manual')}
                    className={`rounded-md px-3 py-1.5 transition-colors ${
                      csrInputMode === 'manual'
                        ? 'border border-blue-500/30 bg-blue-500/10 text-blue-700 shadow-sm dark:text-blue-300'
                        : 'border border-transparent text-[var(--zl-text-muted)] hover:bg-[var(--zl-control-bg)] hover:text-[var(--zl-text)]'
                    }`}
                  >
                    手动提供
                  </button>
                </div>
              </div>
              {csrInputMode === 'manual' ? (
                <div className="mt-4 space-y-3">
                  <label className="block text-xs font-medium text-[var(--zl-text)]">
                    上传 CSR 文件
                    <span className="mt-1.5 flex cursor-pointer items-center gap-2 rounded-lg border border-dashed border-[var(--zl-border)] bg-[var(--zl-control-bg)] px-3 py-2 text-[var(--zl-text-muted)] transition-colors hover:border-blue-500/50 hover:text-blue-500">
                      <Upload size={14} />
                      选择 .csr、.pem 或 .txt 文件
                      <input
                        type="file"
                        accept=".csr,.pem,.txt"
                        className="sr-only"
                        onChange={event => void readCSRFile(event.target.files?.[0])}
                      />
                    </span>
                  </label>
                  <Field label="CSR 内容" required>
                    <textarea
                      value={csrPEM}
                      onChange={event => void updateManualCSR(event.target.value)}
                      rows={7}
                      className="zl-form-control w-full resize-y rounded-xl px-3 py-3 font-mono text-xs leading-5"
                      placeholder="-----BEGIN CERTIFICATE REQUEST-----"
                    />
                  </Field>
                </div>
              ) : null}
            </section>

            <div className="mt-4 grid gap-4 sm:grid-cols-2">
              <Field label="签发 CA" required>
                <Select value={caID} onValueChange={setCAID}>
                  <SelectTrigger className="font-normal data-[placeholder]:text-[var(--zl-text-muted)]">
                    <SelectValue placeholder="选择签发 CA" />
                  </SelectTrigger>
                  <SelectContent>
                    {issuingCAs.map(item => (
                      <SelectItem key={item.id} value={item.id} className="font-normal">
                        {item.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </Field>
              <Field label="用途">
                <Select
                  value={purpose}
                  onValueChange={(value: CertificatePurpose) => setPurpose(value)}
                >
                  <SelectTrigger className="font-normal">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="server" className="font-normal">
                      服务端证书
                    </SelectItem>
                    <SelectItem value="client" className="font-normal">
                      客户端证书
                    </SelectItem>
                    <SelectItem value="mtls" className="font-normal">
                      mTLS 双向认证
                    </SelectItem>
                  </SelectContent>
                </Select>
              </Field>
            </div>

            {issuingCAs.length === 0 && !caQuery.isLoading ? (
              <p className="mt-4 rounded-lg border border-amber-500/20 bg-amber-500/10 px-3 py-2 text-xs leading-5 text-amber-600 dark:text-amber-400">
                当前没有可用签发 CA，请先在 CA 管理中创建
              </p>
            ) : null}
            <div className="mt-5 flex justify-end border-t border-[var(--zl-border)] pt-4">
              <Button
                type="button"
                onClick={() => void goNext()}
                disabled={issuingCAs.length === 0 || isGeneratingCSR}
              >
                {isGeneratingCSR ? '正在生成 CSR...' : '下一步'}
                <ArrowRight size={15} />
              </Button>
            </div>
          </section>
        ) : null}

        {step === 2 ? (
          <RequestPreview
            subject={subject}
            algorithm={algorithm}
            validityLabel={validityLabel}
            caName={selectedCA?.name ?? '-'}
            purpose={purpose}
            san={sanValues}
            subjectText={subjectText}
            csrInputMode={csrInputMode}
            csrPEM={csrPreview?.csrPEM}
            submitting={mutation.isPending}
            onBack={() => setStep(1)}
          />
        ) : null}

        {step === 3 ? (
          <section className="zl-surface-3d certflow-page-card certflow-scroll-area rounded-xl border px-5 py-10 text-center">
            <span className="mx-auto grid h-14 w-14 place-items-center rounded-2xl border border-emerald-500/20 bg-emerald-500/10 text-emerald-500">
              <CheckCircle2 size={27} />
            </span>
            <h2 className="mt-4 text-lg font-semibold">申请已提交</h2>
            <p className="mt-2 text-sm text-[var(--zl-text-muted)]">
              证书申请已进入审批流程，请等待管理员审核
            </p>
            <p className="mt-2 text-xs text-[var(--zl-text-muted)]">申请 CN：{subject.cn || '-'}</p>
            <div className="mt-6 flex justify-center">
              <Button type="button" variant="outline" className="!w-64" onClick={reset}>
                再申请一张
              </Button>
            </div>
          </section>
        ) : null}
      </form>
    </div>
  );
}

function RequestSteps({ step }: { step: Step }) {
  const steps: Array<{ value: Step; label: string }> = [
    { value: 1, label: '填写申请信息' },
    { value: 2, label: '预览与确认' },
    { value: 3, label: '提交完成' },
  ];
  return (
    <ol className="zl-surface-3d flex rounded-xl border px-4 py-3">
      {steps.map((item, index) => (
        <li key={item.value} className="flex min-w-0 flex-1 items-center last:flex-none">
          <div className="flex items-center gap-2 whitespace-nowrap">
            <span
              className={`grid h-7 w-7 place-items-center rounded-full border text-xs font-semibold ${
                step === item.value
                  ? 'border-blue-500 bg-blue-500/10 text-blue-600 dark:text-blue-400'
                  : step > item.value
                    ? 'border-emerald-500/45 bg-emerald-500/12 text-emerald-700 shadow-sm shadow-emerald-500/10 dark:text-emerald-300'
                    : 'border-[var(--zl-border)] bg-[var(--zl-control-bg-soft)] text-[var(--zl-text-muted)]'
              }`}
            >
              {step > item.value ? <CheckCircle2 size={14} /> : item.value}
            </span>
            <span
              className={`hidden text-sm sm:inline ${
                step === item.value
                  ? 'font-medium text-[var(--zl-text)]'
                  : 'text-[var(--zl-text-muted)]'
              }`}
            >
              {item.label}
            </span>
          </div>
          {index < steps.length - 1 ? (
            <span className="mx-3 h-px min-w-3 flex-1 bg-[var(--zl-border)]" aria-hidden="true" />
          ) : null}
        </li>
      ))}
    </ol>
  );
}
