import { ChangeEvent, FormEvent, useEffect, useRef, useState } from 'react'
import { createRoot } from 'react-dom/client'
import './styles.css'

type Item = { case_id: string; hf_type: string; days: number; status: string; sequence?: number }
type Check = { day: number; weight_kg: number; sbp: number; hr: number; spo2: number; edema_grade: number; dyspnea_rest: boolean; pnd: boolean; dyspnea_exertion: string }
type Rule = { rule_id: string; label: string }
type Assessment = { day: number; tier: string; summary: string; fired_rules: Rule[] }
type Detail = { case_id: string; patient: { age: number; sex: string; hf_type: string; lvef_pct: number; baseline: { dry_weight_kg: number; spo2: number } }; checkins: Check[]; assessments: Assessment[]; decision?: { reviewer_tier: string; agreement: string; disagree_note?: string; seconds_spent?: number } }
type Summary = { reviewed_cases: number; total_cases: number; agent_match_rate: number; reviewer_agent_kappa?: number; reviewer_agent_weighted_kappa?: number; median_seconds: number; emergency_sensitivity: number; low_risk_specificity: number }
type LLMStatus = { enabled: boolean; model?: string; message: string }
type EvidenceCitation = { day: number; field: string; value: string; note: string }
type LLMAssessment = { risk_tier: string; confidence: 'low' | 'medium' | 'high'; rules_alignment: 'rules_align' | 'rules_differ' | 'insufficient_evidence'; rationale: string; key_signals: string[]; evidence: EvidenceCitation[]; questions: string[]; safety_note: string; model: string; generated_at: string }
type SafetyMetrics = { evaluated_cases: number; exact_matches: number; exact_match_rate: number; emergency_cases: number; emergency_detected: number; emergency_sensitivity: number; low_risk_cases: number; low_risk_correct: number; low_risk_specificity: number; critical_misses: number; evidence_backed_cases: number }
type SafetyReport = { dataset_cases: number; rules: SafetyMetrics; model: SafetyMetrics; clinicians: SafetyMetrics; benchmark: { status: string; model?: string; total: number; completed: number; error?: string }; disagreements: { case_id: string; reviewer_code: string; reviewer_tier: string; rule_tier: string; model_tier?: string; agreement: string; note?: string }[]; safety_note: string }
type LabCopy = { nav: string; title: string; subtitle: string; run: string; running: string; configured: string; notConfigured: string; rules: string; model: string; clinicians: string; coverage: string; exact: string; l3: string; lowRisk: string; cited: string; disagreements: string; noDisagreements: string; modelRun: string; status: string; safety: string; loading: string; confidence: string; alignment: string; evidence: string; rulesAlign: string; rulesDiffer: string; insufficient: string }
type Pref = { locale: 'zh' | 'en'; theme: 'light' | 'dark'; logoData: string; logoName: string; maskCaseIDs: boolean; confirmSave: boolean; hideMetrics: boolean; autoLock: number }

const api = 'http://localhost:8080'
const defaults: Pref = { locale: 'zh', theme: 'light', logoData: '', logoName: '', maskCaseIDs: false, confirmSave: true, hideMetrics: false, autoLock: 15 }
const words = {
  zh: {
    workspace: '審閱工作台', settings: '設定', study: 'HF 出院後追蹤研究', queue: '個案佇列', cases: '筆個案', pending: '待審閱', completed: '已完成', reviewer: '審閱者', research: '研究工作區', overview: '研究概覽', patient: '病人摘要', trend: '臨床趨勢', timeline: '每日追蹤', evidence: '觸發證據', decision: '審閱決策', tier: '最終分級', assessment: '與系統判讀', note: '判讀備註', save: '儲存審閱決策', saving: '儲存中…', saved: '決策已儲存', empty: '從左側佇列選取個案以開始審閱', agent: '系統建議', current: '最新狀態', weight: '體重', oxygen: '血氧', baseline: '基準值', risk: '風險分級', security: '資訊安全與隱私', appearance: '外觀與品牌', language: '介面語言', theme: '顯示模式', light: '亮色', dark: '暗色', logo: '機構 Logo', upload: '上傳圖片', remove: '移除 Logo', logoHint: '支援 PNG、JPG、SVG；上限 1.5 MB。圖片只儲存在此瀏覽器。', mask: '在共用螢幕遮蔽個案識別碼', confirm: '儲存決策前要求確認', hide: '在工作區隱藏研究指標', lock: '閒置自動鎖定', off: '關閉', minutes: '分鐘', local: '偏好設定僅儲存在目前瀏覽器，並不取代組織的身份驗證與存取控管。', locked: '工作區已鎖定', unlock: '繼續工作', lockText: '為保護畫面中的研究資訊，請解鎖後繼續。', error: '無法載入研究資料。請確認 API 已啟動。', synthetic: '僅限合成研究資料，非臨床決策支援。', csv: 'CSV', json: 'JSON', noEvidence: '此追蹤日未觸發升級規則。', agree: '同意', modify: '修改', disagree: '不同意', confirmSave: '確定儲存這筆審閱決策？', uploadError: '請選擇小於 1.5 MB 的 PNG、JPG 或 SVG 圖片。', reviewed: '已審閱', metrics: '研究指標', sensitivity: '緊急敏感度', specificity: '低風險特異度', median: '中位審閱時間', weighted: '加權 κ', llm: 'LLM 研究輔助', llmDescription: '使用伺服器端設定的模型，針對合成研究個案提供可追溯的輔助摘要。', llmEnabled: '已在伺服器端啟用', llmDisabled: '尚未設定 LLM 連線', llmSecurity: 'API Key 只保留在伺服器環境變數；不會傳送到瀏覽器或存入研究資料庫。', llmRun: '產生研究輔助摘要', llmRunning: '正在產生摘要…', llmSignals: '關鍵訊號', llmQuestions: '建議釐清事項', llmResearchOnly: '僅供合成研究輔助，醫師須獨立判斷。'
  },
  en: {
    workspace: 'Review workspace', settings: 'Settings', study: 'HF post-discharge reader study', queue: 'Case queue', cases: 'cases', pending: 'Pending', completed: 'Complete', reviewer: 'Reviewer', research: 'Research workspace', overview: 'Study overview', patient: 'Patient snapshot', trend: 'Clinical trend', timeline: 'Daily follow-up', evidence: 'Triggered evidence', decision: 'Reviewer decision', tier: 'Final tier', assessment: 'Assessment alignment', note: 'Review note', save: 'Save review decision', saving: 'Saving…', saved: 'Decision saved', empty: 'Select a case from the queue to begin review', agent: 'Agent recommendation', current: 'Current status', weight: 'Weight', oxygen: 'Oxygen', baseline: 'Baseline', risk: 'Risk tier', security: 'Security & privacy', appearance: 'Appearance & brand', language: 'Interface language', theme: 'Display mode', light: 'Light', dark: 'Dark', logo: 'Organisation logo', upload: 'Upload image', remove: 'Remove logo', logoHint: 'PNG, JPG, or SVG up to 1.5 MB. The image remains only in this browser.', mask: 'Mask case identifiers on shared screens', confirm: 'Require confirmation before saving a decision', hide: 'Hide study metrics in the workspace', lock: 'Auto-lock when inactive', off: 'Off', minutes: 'minutes', local: 'Preferences are stored only in this browser and do not replace organisational authentication or access controls.', locked: 'Workspace locked', unlock: 'Continue working', lockText: 'Unlock to continue viewing research information.', error: 'Could not load study data. Is the API running?', synthetic: 'Synthetic research data only. Not clinical decision support.', csv: 'CSV', json: 'JSON', noEvidence: 'No escalation rules were triggered for this check-in.', agree: 'Agree', modify: 'Modify', disagree: 'Disagree', confirmSave: 'Save this reviewer decision?', uploadError: 'Choose a PNG, JPG, or SVG image smaller than 1.5 MB.', reviewed: 'Reviewed', metrics: 'Study metrics', sensitivity: 'Emergency sensitivity', specificity: 'Low-risk specificity', median: 'Median review', weighted: 'Weighted κ', llm: 'LLM research assistant', llmDescription: 'Uses a server-configured model to provide a traceable supporting summary for synthetic study cases.', llmEnabled: 'Enabled on the server', llmDisabled: 'LLM connection is not configured', llmSecurity: 'The API key remains in server environment variables and is never sent to the browser or stored in the study database.', llmRun: 'Generate research summary', llmRunning: 'Generating summary…', llmSignals: 'Key signals', llmQuestions: 'Questions to clarify', llmResearchOnly: 'Synthetic research assistance only. Clinicians must apply independent judgment.'
  }
} as const

function readPrefs(): Pref {
  try {
    const saved = JSON.parse(localStorage.getItem('hf-readmit-settings') || '{}')
    return { ...defaults, ...saved, locale: saved.locale === 'en' ? 'en' : saved.locale === 'zh' ? 'zh' : defaults.locale, logoData: saved.logoData || '', logoName: saved.logoName || '' }
  } catch { return defaults }
}
function pct(value?: number) { return value == null ? 'N/A' : Math.round(value * 100) + '%' }
function maskID(value: string, masked: boolean) { return masked ? 'Case ···' + value.slice(-3) : value }
function Tier({ value, compact = false }: { value: string; compact?: boolean }) { return <span className={'tier ' + value + (compact ? ' compact' : '')}>{value}</span> }
function Icon({ name }: { name: 'workspace' | 'settings' | 'sun' | 'moon' | 'lock' | 'download' | 'shield' | 'panel' | 'patient' | 'scale' | 'oxygen' | 'pulse' }) {
  const common = { fill: 'none', stroke: 'currentColor', strokeWidth: 1.8, strokeLinecap: 'round' as const, strokeLinejoin: 'round' as const }
  if (name === 'settings') return <svg viewBox="0 0 24 24" aria-hidden="true"><circle cx="12" cy="12" r="3" {...common}/><path d="M19.4 15a1.7 1.7 0 0 0 .34 1.88l.06.06-2.1 2.1-.06-.06A1.7 1.7 0 0 0 15.76 19a1.7 1.7 0 0 0-1.03 1.55V21h-3v-.45A1.7 1.7 0 0 0 10.7 19a1.7 1.7 0 0 0-1.88.34l-.06.06-2.1-2.1.06-.06A1.7 1.7 0 0 0 7 15.36a1.7 1.7 0 0 0-1.55-1.03H5v-3h.45A1.7 1.7 0 0 0 7 10.3a1.7 1.7 0 0 0-.34-1.88L6.6 8.36l2.1-2.1.06.06A1.7 1.7 0 0 0 10.64 6a1.7 1.7 0 0 0 1.03-1.55V4h3v.45A1.7 1.7 0 0 0 15.7 6a1.7 1.7 0 0 0 1.88-.34l.06-.06 2.1 2.1-.06.06A1.7 1.7 0 0 0 19 9.64a1.7 1.7 0 0 0 1.55 1.03H21v3h-.45A1.7 1.7 0 0 0 19.4 15Z" {...common}/></svg>
  if (name === 'sun') return <svg viewBox="0 0 24 24" aria-hidden="true"><circle cx="12" cy="12" r="4" {...common}/><path d="M12 2v2m0 16v2M4.93 4.93l1.41 1.41m11.32 11.32 1.41 1.41M2 12h2m16 0h2M4.93 19.07l1.41-1.41M17.66 6.34l1.41-1.41" {...common}/></svg>
  if (name === 'moon') return <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M20.6 15.4A8.7 8.7 0 0 1 8.6 3.4 8.7 8.7 0 1 0 20.6 15.4Z" {...common}/></svg>
  if (name === 'lock') return <svg viewBox="0 0 24 24" aria-hidden="true"><rect x="5" y="10" width="14" height="11" rx="2" {...common}/><path d="M8 10V7a4 4 0 0 1 8 0v3" {...common}/></svg>
  if (name === 'download') return <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M12 3v12m0 0 4-4m-4 4-4-4M4 20h16" {...common}/></svg>
  if (name === 'shield') return <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M12 3 5 6v5c0 4.8 2.8 8.4 7 10 4.2-1.6 7-5.2 7-10V6l-7-3Z" {...common}/><path d="m9 12 2 2 4-4" {...common}/></svg>
  if (name === 'panel') return <svg viewBox="0 0 24 24" aria-hidden="true"><rect x="4" y="4" width="16" height="16" rx="3" {...common}/><path d="M9 4v16M13 9h4M13 13h4" {...common}/></svg>
  if (name === 'patient') return <svg viewBox="0 0 24 24" aria-hidden="true"><circle cx="12" cy="8" r="3.2" {...common}/><path d="M5.5 20c.8-3.5 3.2-5.4 6.5-5.4s5.7 1.9 6.5 5.4" {...common}/></svg>
  if (name === 'scale') return <svg viewBox="0 0 24 24" aria-hidden="true"><rect x="4" y="5" width="16" height="15" rx="3" {...common}/><path d="M8 10a4 4 0 0 1 8 0M12 10l2.1-1.4" {...common}/></svg>
  if (name === 'oxygen') return <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M12 3s-5 5.5-5 9.6a5 5 0 0 0 10 0C17 8.5 12 3 12 3Z" {...common}/><path d="M9.5 14c.4 1.2 1.2 1.9 2.5 2.1" {...common}/></svg>
  if (name === 'pulse') return <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M3 12h4l2-5 4 10 2-5h6" {...common}/></svg>
  return <svg viewBox="0 0 24 24" aria-hidden="true"><rect x="4" y="4" width="16" height="16" rx="3" {...common}/><path d="M8 9h8M8 13h8M8 17h4" {...common}/></svg>
}
function Trend({ rows }: { rows: Check[] }) {
  const points = (key: 'weight_kg' | 'spo2') => {
    const values = rows.map(row => row[key]); const min = Math.min(...values); const max = Math.max(...values); const range = Math.max(max - min, 1)
    return rows.map((row, index) => 26 + index * 608 / Math.max(rows.length - 1, 1) + ',' + (126 - (row[key] - min) / range * 84)).join(' ')
  }
  return <figure className="trend-chart"><div className="chart-legend"><span className="weight">Weight</span><span className="oxygen">Oxygen</span><span className="chart-note">independent scales</span></div><svg viewBox="0 0 660 150" role="img" aria-label="Weight and oxygen trend"><path d="M26 22H634M26 64H634M26 106H634" className="chart-grid"/><polyline className="line-weight" points={points('weight_kg')}/><polyline className="line-oxygen" points={points('spo2')}/>{rows.map((row, index) => <g key={row.day}><circle className="dot-weight" cx={26 + index * 608 / Math.max(rows.length - 1, 1)} cy={Number(points('weight_kg').split(' ')[index].split(',')[1])} r="3"/><text x={26 + index * 608 / Math.max(rows.length - 1, 1)} y="145" textAnchor="middle">D{row.day}</text></g>)}</svg></figure>
}
function SafetyMetricCard({ title, metric, total, evidenceLabel }: { title: string; metric: SafetyMetrics; total: number; evidenceLabel?: string }) {
  return <article className="safety-metric-card"><div className="safety-card-heading"><h3>{title}</h3><span>{metric.evaluated_cases}/{total}</span></div><div className="safety-primary"><span>Exact match</span><b>{pct(metric.exact_match_rate)}</b></div><div className="safety-metric-grid"><div><span>L3 recall</span><b>{pct(metric.emergency_sensitivity)}</b></div><div><span>Low-risk</span><b>{pct(metric.low_risk_specificity)}</b></div><div><span>Critical misses</span><b className={metric.critical_misses ? 'critical' : ''}>{metric.critical_misses}</b></div>{evidenceLabel && <div><span>{evidenceLabel}</span><b>{metric.evidence_backed_cases}/{metric.evaluated_cases}</b></div>}</div></article>
}

function SafetyLabView({ report, copy, loading, enabled, model, onRun, error }: { report: SafetyReport | null; copy: LabCopy; loading: boolean; enabled: boolean; model?: string; onRun: () => void; error: string }) {
  const running = report?.benchmark.status === 'running'
  return <section className="safety-page"><header className="safety-intro"><div><p className="eyebrow">{copy.nav}</p><h2>{copy.title}</h2><p>{copy.subtitle}</p></div><Icon name="shield"/></header><section className="safety-control"><span className={'lab-dot ' + (enabled ? 'enabled' : '')}></span><div><b>{enabled ? copy.configured : copy.notConfigured}</b><small>{report?.benchmark.model || model || copy.modelRun}</small></div><button className="primary-button" type="button" disabled={!enabled || loading || running} onClick={onRun}>{running ? copy.running + ' ' + report?.benchmark.completed + '/' + report?.benchmark.total : copy.run}</button></section>{error && <p className="lab-error" role="alert">{error}</p>}{!report ? <div className="safety-empty">{copy.loading}</div> : <><div className="safety-grid"><SafetyMetricCard title={copy.rules} metric={report.rules} total={report.dataset_cases}/><SafetyMetricCard title={copy.model} metric={report.model} total={report.dataset_cases} evidenceLabel={copy.cited}/><SafetyMetricCard title={copy.clinicians} metric={report.clinicians} total={report.dataset_cases}/></div><section className="lab-panel"><div className="lab-panel-heading"><div><p className="eyebrow">{copy.status}</p><h3>{copy.modelRun}</h3></div><span className={'benchmark-status ' + report.benchmark.status}>{report.benchmark.status}</span></div><div className="benchmark-grid"><div><span>{copy.coverage}</span><b>{report.model.evaluated_cases}/{report.dataset_cases}</b></div><div><span>{copy.cited}</span><b>{report.model.evidence_backed_cases}/{report.model.evaluated_cases}</b></div><div><span>{copy.safety}</span><b>{report.model.critical_misses}</b></div></div>{report.benchmark.error && <p className="lab-error">{report.benchmark.error}</p>}</section><section className="lab-panel"><div className="lab-panel-heading"><div><p className="eyebrow">{copy.disagreements}</p><h3>{copy.disagreements}</h3></div><span>{report.disagreements.length}</span></div>{report.disagreements.length ? <div className="disagreement-list">{report.disagreements.map(item => <article key={item.case_id + item.reviewer_code}><div><b>{item.case_id}</b><span>{item.reviewer_code} · {item.agreement}</span></div><div className="tier-compare"><span>Human <Tier value={item.reviewer_tier} compact/></span><span>Rules <Tier value={item.rule_tier} compact/></span>{item.model_tier && <span>GPT <Tier value={item.model_tier} compact/></span>}</div>{item.note && <p>{item.note}</p>}</article>)}</div> : <p className="no-evidence">{copy.noDisagreements}</p>}</section><p className="lab-footnote"><Icon name="shield"/>{report.safety_note}</p></>}</section>
}

function App() {
  const [prefs, setPrefs] = useState<Pref>(readPrefs)
  const [page, setPage] = useState<'review' | 'settings' | 'safety'>('review')
  const [reviewer, setReviewer] = useState('R1')
  const [items, setItems] = useState<Item[]>([])
  const [detail, setDetail] = useState<Detail | null>(null)
  const [summary, setSummary] = useState<Summary | null>(null)
  const [tier, setTier] = useState('L0')
  const [agreement, setAgreement] = useState('agree')
  const [note, setNote] = useState('')
  const [message, setMessage] = useState('')
  const [error, setError] = useState('')
  const [saving, setSaving] = useState(false)
  const [locked, setLocked] = useState(false)
  const [activityAt, setActivityAt] = useState(Date.now())
  const [settingsMessage, setSettingsMessage] = useState('')
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false)
  const [llmStatus, setLLMStatus] = useState<LLMStatus | null>(null)
  const [llmAssessment, setLLMAssessment] = useState<LLMAssessment | null>(null)
  const [llmLoading, setLLMLoading] = useState(false)
  const [llmError, setLLMError] = useState('')
  const [safetyReport, setSafetyReport] = useState<SafetyReport | null>(null)
  const [safetyLoading, setSafetyLoading] = useState(false)
  const [safetyError, setSafetyError] = useState('')
  const uploadRef = useRef<HTMLInputElement>(null)
  const t = words[prefs.locale]
  const labels = prefs.locale === 'zh' ? { preferences: '工作區偏好', preferencesCopy: '管理此裝置上的研究工作區外觀與資訊保護方式。', appearanceCopy: '語言、顯示模式與本機品牌識別。', securityCopy: '控制此瀏覽器如何顯示研究資料。', trendTitle: '體重與血氧飽和度', trailTitle: '每日判讀紀錄', rationaleTitle: '規則依據', checkins: '次追蹤', flagged: '個觸發日', ageSex: '年齡／性別' } : { preferences: 'Workspace preferences', preferencesCopy: 'Manage how this research workspace appears and protects information on this device.', appearanceCopy: 'Language, display mode, and local branding.', securityCopy: 'Control how research details are exposed on this browser.', trendTitle: 'Weight & oxygenation', trailTitle: 'Daily review trail', rationaleTitle: 'Rule rationale', checkins: 'check-ins', flagged: 'flagged days', ageSex: 'Age / sex' }
  const headers = { 'X-Reviewer-Code': reviewer }
  const updatePrefs = (patch: Partial<Pref>) => setPrefs(previous => ({ ...previous, ...patch }))

  useEffect(() => { try { localStorage.setItem('hf-readmit-settings', JSON.stringify(prefs)) } catch { setSettingsMessage('Unable to store this browser preference.') } }, [prefs])
  useEffect(() => {
    if (!prefs.autoLock) return
    const noteActivity = () => { if (!locked) setActivityAt(Date.now()) }
    window.addEventListener('mousemove', noteActivity); window.addEventListener('keydown', noteActivity); window.addEventListener('touchstart', noteActivity)
    const timer = window.setInterval(() => { if (Date.now() - activityAt >= prefs.autoLock * 60000) setLocked(true) }, 10000)
    return () => { window.removeEventListener('mousemove', noteActivity); window.removeEventListener('keydown', noteActivity); window.removeEventListener('touchstart', noteActivity); window.clearInterval(timer) }
  }, [prefs.autoLock, activityAt, locked])
  const load = async () => {
    const queue = await fetch(api + '/api/cases', { headers })
    const analytics = await fetch(api + '/api/analytics/summary')
    if (!queue.ok || !analytics.ok) throw new Error()
    setItems(await queue.json()); setSummary(await analytics.json())
  }
  const loadSafety = async () => {
    const response = await fetch(api + '/api/safety-lab')
    if (!response.ok) throw new Error()
    setSafetyReport(await response.json())
  }
  useEffect(() => { setDetail(null); setError(''); setLLMAssessment(null); setLLMError(''); load().catch(() => setError(t.error)) }, [reviewer])
  useEffect(() => { fetch(api + '/api/llm/status').then(response => response.ok ? response.json() : null).then(setLLMStatus).catch(() => setLLMStatus(null)) }, [])
  useEffect(() => { if (page === 'safety') loadSafety().catch(() => setSafetyError('Unable to load safety evaluation.')) }, [page])
  useEffect(() => { if (page !== 'safety' || safetyReport?.benchmark.status !== 'running') return; const timer = window.setInterval(() => { loadSafety().catch(() => setSafetyError('Unable to refresh safety evaluation.')) }, 2000); return () => window.clearInterval(timer) }, [page, safetyReport?.benchmark.status])
  const select = async (caseID: string) => {
    setError(''); setMessage(''); setLLMAssessment(null); setLLMError('')
    try {
      await fetch(api + '/api/cases/' + caseID + '/open', { method: 'POST', headers })
      const response = await fetch(api + '/api/cases/' + caseID, { headers })
      if (!response.ok) throw new Error()
      const next: Detail = await response.json()
      setDetail(next); setTier(next.decision?.reviewer_tier || next.assessments.at(-1)?.tier || 'L0'); setAgreement(next.decision?.agreement || 'agree'); setNote(next.decision?.disagree_note || '')
    } catch { setError(t.error) }
  }
  const submit = async (event: FormEvent) => {
    event.preventDefault(); if (!detail || saving) return
    if (prefs.confirmSave && !window.confirm(t.confirmSave)) return
    setSaving(true); setError('')
    try {
      const response = await fetch(api + '/api/cases/' + detail.case_id + '/decision', { method: 'POST', headers: { ...headers, 'Content-Type': 'application/json' }, body: JSON.stringify({ reviewer_tier: tier, agreement, disagree_note: note, seconds_spent: 0 }) })
      if (!response.ok) { const body = await response.json(); throw new Error(body.error) }
      await load(); await select(detail.case_id); setMessage(t.saved)
    } catch (reason) { setError(reason instanceof Error && reason.message || t.error) } finally { setSaving(false) }
  }
  const requestLLMReview = async () => {
    if (!detail || llmLoading || !llmStatus?.enabled) return
    setLLMLoading(true); setLLMError(''); setLLMAssessment(null)
    try {
      const response = await fetch(api + '/api/cases/' + detail.case_id + '/llm-assessment', { method: 'POST', headers })
      const body = await response.json()
      if (!response.ok) { throw new Error(body.error || body.message || t.llmDisabled) }
      setLLMAssessment(body)
    } catch (reason) { setLLMError(reason instanceof Error ? reason.message : t.llmDisabled) } finally { setLLMLoading(false) }
  }
  const runBenchmark = async () => {
    if (!llmStatus?.enabled || safetyLoading) return
    setSafetyLoading(true); setSafetyError('')
    try { const response = await fetch(api + '/api/safety-lab/benchmark', { method: 'POST' }); const body = await response.json(); if (!response.ok) throw new Error(body.error || body.message || 'Unable to start benchmark.'); setSafetyReport(body) } catch (reason) { setSafetyError(reason instanceof Error ? reason.message : 'Unable to start benchmark.') } finally { setSafetyLoading(false) }
  }
  const jumpToDay = (day: number) => document.getElementById('day-' + day)?.scrollIntoView({ behavior: 'smooth', block: 'center' })
  const uploadLogo = (event: ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0]; if (!file) return
    if (!file.type.startsWith('image/') || file.size > 1500000) { setSettingsMessage(t.uploadError); return }
    const reader = new FileReader()
    reader.onload = () => { updatePrefs({ logoData: String(reader.result), logoName: file.name }); setSettingsMessage('') }
    reader.readAsDataURL(file)
  }
  const latest = detail?.assessments.at(-1)
  const latestCheck = detail?.checkins.at(-1)
  const clinical = prefs.locale === 'zh'
    ? { latest: '\u6700\u65b0\u8ffd\u8e64', weightChange: '\u8207\u4e7e\u91cd\u5dee\u7570', stable: '\u5e73\u7a69', sbp: '\u6536\u7e2e\u58d3', heartRate: '\u5fc3\u7387', edema: '\u6c34\u8173', grade: '\u7d1a' }
    : { latest: 'Latest check-in', weightChange: 'vs dry weight', stable: 'Stable', sbp: 'Systolic BP', heartRate: 'Heart rate', edema: 'Edema', grade: 'grade' }
  const weightChange = detail && latestCheck ? latestCheck.weight_kg - detail.patient.baseline.dry_weight_kg : 0
  const weightTrend = Math.abs(weightChange) < 0.1 ? clinical.stable : `${weightChange > 0 ? '+' : ''}${weightChange.toFixed(1)} kg`
  const weightState = weightChange >= 2 ? ' elevated' : weightChange <= -2 ? ' reduced' : ''
  const lab: LabCopy = prefs.locale === 'zh'
    ? { nav: '\u5b89\u5168\u8a55\u6e2c\u5be6\u9a57\u5ba4', title: '\u8b49\u64da\u5c0e\u5411 AI \u5b89\u5168\u8a55\u6e2c', subtitle: '\u6bd4\u8f03\u898f\u5247\u3001GPT \u8207\u91ab\u5e2b\u5728\u5408\u6210\u7814\u7a76\u500b\u6848\u4e2d\u7684\u5224\u8b80\u3002', run: '\u57f7\u884c 18 \u500b\u6848\u4f8b GPT \u8a55\u6e2c', running: '\u6b63\u5728\u8a55\u6e2c', configured: 'GPT \u7814\u7a76\u8f14\u52a9\u5df2\u555f\u7528', notConfigured: 'GPT \u7814\u7a76\u8f14\u52a9\u672a\u8a2d\u5b9a', rules: '\u78ba\u5b9a\u6027\u898f\u5247', model: 'GPT \u7b2c\u4e8c\u610f\u898b', clinicians: '\u91ab\u5e2b\u5be9\u95b1', coverage: '\u8a55\u6e2c\u8986\u84cb\u7387', exact: '\u6b63\u78ba\u7387', l3: 'L3 \u654f\u611f\u5ea6', lowRisk: '\u4f4e\u98a8\u96aa\u7279\u7570\u5ea6', cited: '\u8b49\u64da\u5f15\u7528', disagreements: '\u91ab\u5e2b\u5206\u6b67\u4e32\u5217', noDisagreements: '\u76ee\u524d\u6c92\u6709\u5df2\u5132\u5b58\u7684\u5be9\u95b1\u5206\u6b67\u3002', modelRun: '\u6a21\u578b\u8a55\u6e2c\u57f7\u884c\u72c0\u614b', status: '\u57f7\u884c\u72c0\u614b', safety: '\u95dc\u9375\u6f0f\u5224', loading: '\u6b63\u5728\u8f09\u5165\u5b89\u5168\u8a55\u6e2c\u2026', confidence: '\u4fe1\u5fc3\u7a0b\u5ea6', alignment: '\u8207\u898f\u5247\u4e00\u81f4\u6027', evidence: '\u53ef\u9a57\u8b49\u8b49\u64da', rulesAlign: '\u8207\u898f\u5247\u4e00\u81f4', rulesDiffer: '\u8207\u898f\u5247\u4e0d\u540c', insufficient: '\u8b49\u64da\u4e0d\u8db3' }
    : { nav: 'Safety Lab', title: 'Evidence-first AI safety evaluation', subtitle: 'Compare deterministic rules, GPT, and clinician decisions across synthetic study cases.', run: 'Run GPT benchmark on 18 cases', running: 'Evaluating', configured: 'GPT research assistant enabled', notConfigured: 'GPT research assistant not configured', rules: 'Deterministic rules', model: 'GPT second reader', clinicians: 'Clinician review', coverage: 'Evaluation coverage', exact: 'Exact match', l3: 'L3 sensitivity', lowRisk: 'Low-risk specificity', cited: 'Evidence cited', disagreements: 'Clinician disagreement trail', noDisagreements: 'No saved clinician disagreements yet.', modelRun: 'Model evaluation run', status: 'Run status', safety: 'Critical misses', loading: 'Loading safety evaluation…', confidence: 'Confidence', alignment: 'Rule alignment', evidence: 'Verifiable evidence', rulesAlign: 'Aligns with rules', rulesDiffer: 'Differs from rules', insufficient: 'Insufficient evidence' }
  const evidence = detail?.assessments.filter(assessment => assessment.fired_rules.length) || []
  const pending = items.filter(item => item.status !== 'reviewed').length
  const completed = items.length - pending
  const showID = (id: string) => maskID(id, prefs.maskCaseIDs)
  return <main className={'app-shell ' + prefs.theme + (sidebarCollapsed ? ' sidebar-collapsed' : '')}>
    <aside className="sidebar">
      <button className="sidebar-collapse" type="button" onClick={() => setSidebarCollapsed(value => !value)} aria-label={sidebarCollapsed ? 'Expand navigation' : 'Collapse navigation'}><Icon name="panel"/></button>
      <div className="brand">{prefs.logoData ? <span className="brand-logo"><img src={prefs.logoData} alt={prefs.logoName || 'Organisation logo'}/></span> : <span className="brand-mark"><span></span><span></span><span></span></span>}<div className="brand-copy"><b>HF Readmit</b><small>Reader study</small></div></div>
      <nav className="primary-nav"><p>{t.research}</p><button className={page === 'review' ? 'active' : ''} onClick={() => setPage('review')}><Icon name="workspace"/>{t.workspace}</button><button className={page === 'safety' ? 'active' : ''} onClick={() => setPage('safety')}><Icon name="shield"/>{lab.nav}</button><button className={page === 'settings' ? 'active' : ''} onClick={() => setPage('settings')}><Icon name="settings"/>{t.settings}</button></nav>
      <div className="sidebar-bottom"><div className="reviewer-card"><span>{t.reviewer}</span><select value={reviewer} onChange={event => setReviewer(event.target.value)}><option>R1</option><option>R2</option></select></div><div className="privacy-note"><Icon name="shield"/><span>{t.synthetic}</span></div></div>
    </aside>
    <section className="app-area">
      <header className="topbar"><div><p>{t.study}</p><h1>{page === 'review' ? t.workspace : page === 'safety' ? lab.nav : t.settings}</h1></div><div className="top-actions"><button className="theme-toggle" onClick={() => updatePrefs({ theme: prefs.theme === 'light' ? 'dark' : 'light' })} aria-label="Toggle theme"><Icon name={prefs.theme === 'light' ? 'moon' : 'sun'}/></button>{page === 'review' && <div className="exports"><Icon name="download"/><a href={api + '/api/analytics/export?format=csv'}>{t.csv}</a><a href={api + '/api/analytics/export?format=json'}>{t.json}</a></div>}</div></header>
      {error && <div className="alert" role="alert">{error}</div>}
      {page === 'settings' ? <section className="settings-page">
        <div className="settings-intro"><div><p className="eyebrow">{t.settings}</p><h2>{labels.preferences}</h2><p>{labels.preferencesCopy}</p></div><Icon name="settings"/></div>
        <div className="settings-grid">
          <article className="settings-card"><div className="card-heading"><div><Icon name="workspace"/><div><h3>{t.appearance}</h3><p>{labels.appearanceCopy}</p></div></div></div>
            <label>{t.language}<select value={prefs.locale} onChange={event => updatePrefs({ locale: event.target.value as Pref['locale'] })}><option value="zh">繁體中文</option><option value="en">English</option></select></label>
            <div className="theme-choice"><span>{t.theme}</span><div><button className={prefs.theme === 'light' ? 'selected' : ''} onClick={() => updatePrefs({ theme: 'light' })}><Icon name="sun"/>{t.light}</button><button className={prefs.theme === 'dark' ? 'selected' : ''} onClick={() => updatePrefs({ theme: 'dark' })}><Icon name="moon"/>{t.dark}</button></div></div>
          </article>
          <article className="settings-card"><div className="card-heading"><div><span className="logo-preview">{prefs.logoData ? <img src={prefs.logoData} alt="Logo preview"/> : <span className="brand-mark"><span></span><span></span><span></span></span>}</span><div><h3>{t.logo}</h3><p>{t.logoHint}</p></div></div></div>
            <input ref={uploadRef} type="file" accept="image/png,image/jpeg,image/svg+xml" onChange={uploadLogo} hidden/>
            <div className="logo-actions"><button className="secondary-button" onClick={() => uploadRef.current?.click()}>{t.upload}</button>{prefs.logoData && <button className="text-button" onClick={() => updatePrefs({ logoData: '', logoName: '' })}>{t.remove}</button>}</div>{settingsMessage && <p className="inline-error">{settingsMessage}</p>}
          </article>
          <article className="settings-card llm-card"><div className="card-heading"><div><Icon name="shield"/><div><h3>{t.llm}</h3><p>{t.llmDescription}</p></div></div></div>
            <div className={'llm-status ' + (llmStatus?.enabled ? 'enabled' : 'disabled')}><span></span><div><b>{llmStatus?.enabled ? t.llmEnabled : t.llmDisabled}</b><small>{llmStatus?.enabled ? llmStatus.model : llmStatus?.message}</small></div></div>
            <p className="llm-security-copy"><Icon name="lock"/>{t.llmSecurity}</p>
          </article>
          <article className="settings-card security-card"><div className="card-heading"><div><Icon name="shield"/><div><h3>{t.security}</h3><p>{labels.securityCopy}</p></div></div></div>
            <label className="switch-row"><span>{t.mask}</span><input type="checkbox" checked={prefs.maskCaseIDs} onChange={event => updatePrefs({ maskCaseIDs: event.target.checked })}/><i></i></label>
            <label className="switch-row"><span>{t.confirm}</span><input type="checkbox" checked={prefs.confirmSave} onChange={event => updatePrefs({ confirmSave: event.target.checked })}/><i></i></label>
            <label className="switch-row"><span>{t.hide}</span><input type="checkbox" checked={prefs.hideMetrics} onChange={event => updatePrefs({ hideMetrics: event.target.checked })}/><i></i></label>
            <label className="select-row"><span>{t.lock}</span><select value={prefs.autoLock} onChange={event => updatePrefs({ autoLock: Number(event.target.value) })}><option value="0">{t.off}</option><option value="5">5 {t.minutes}</option><option value="15">15 {t.minutes}</option><option value="30">30 {t.minutes}</option></select></label>
            <p className="security-copy"><Icon name="lock"/>{t.local}</p>
          </article>
        </div>
      </section> : page === 'safety' ? <SafetyLabView report={safetyReport} copy={lab} loading={safetyLoading} enabled={Boolean(llmStatus?.enabled)} model={llmStatus?.model} onRun={runBenchmark} error={safetyError}/> : <section className="review-page">
        {!prefs.hideMetrics && <div className="metric-strip"><div><span>{t.metrics}</span><b>{summary?.reviewed_cases || 0} <small>/ {summary?.total_cases || 0}</small></b></div><div><span>{t.weighted}</span><b>{summary?.reviewer_agent_weighted_kappa?.toFixed(2) || 'N/A'}</b></div><div><span>{t.median}</span><b>{summary?.median_seconds ? Math.round(summary.median_seconds) + 's' : 'N/A'}</b></div><div><span>{t.sensitivity}</span><b>{pct(summary?.emergency_sensitivity)}</b></div><div><span>{t.specificity}</span><b>{pct(summary?.low_risk_specificity)}</b></div></div>}
        <div className="workspace-grid">
          <aside className="case-queue"><div className="queue-header"><div><p className="eyebrow">{t.queue}</p><h2>{pending} {t.pending}</h2></div><span>{completed}/{items.length}</span></div><div className="queue-list">{items.map(item => <button className={'queue-item ' + (detail?.case_id === item.case_id ? 'selected' : '')} key={item.case_id} onClick={() => select(item.case_id)}><div><b>{showID(item.case_id)}</b><span>{item.hf_type} · {item.days} days</span></div><div><span className={'queue-dot ' + item.status}></span><small>{item.status === 'reviewed' ? t.reviewed : t.pending}</small></div></button>)}</div></aside>
          <section className="case-detail">{detail ? <><section className="patient-overview" aria-label={t.patient}><header className="patient-summary-header"><div className="patient-identity"><span className="patient-avatar"><Icon name="patient"/></span><div><p className="eyebrow">{showID(detail.case_id)} · {detail.patient.hf_type}</p><h2>{t.patient}</h2><div className="patient-tags"><span>{labels.ageSex} {detail.patient.age} / {detail.patient.sex}</span><span>LVEF {detail.patient.lvef_pct}%</span></div></div></div><div className="status-group"><span>{clinical.latest} · D{latestCheck?.day ?? '—'}</span><Tier value={latest?.tier || 'L0'}/></div></header><div className="clinical-vitals"><article className={'vital-card weight' + weightState}><div className="vital-icon"><Icon name="scale"/></div><div><span>{t.weight}</span><strong>{latestCheck?.weight_kg ?? '—'}<small> kg</small></strong><p>{clinical.weightChange}<b>{weightTrend}</b></p></div></article><article className="vital-card oxygen"><div className="vital-icon"><Icon name="oxygen"/></div><div><span>{t.oxygen}</span><strong>{latestCheck?.spo2 ?? '—'}<small>%</small></strong><p>{t.baseline}<b>{detail.patient.baseline.spo2}%</b></p></div></article><article className="vital-card pulse"><div className="vital-icon"><Icon name="pulse"/></div><div><span>{clinical.sbp}</span><strong>{latestCheck?.sbp ?? '—'}<small> mmHg</small></strong><p>{clinical.heartRate}<b>{latestCheck?.hr ?? '—'} bpm</b></p></div></article><article className="vital-card edema"><div className="vital-icon"><Icon name="patient"/></div><div><span>{clinical.edema}</span><strong>{latestCheck?.edema_grade ?? '—'}<small> {clinical.grade}</small></strong><p>{t.current}<b>{clinical.latest}</b></p></div></article></div></section>            <section className="clinical-section"><div className="section-heading"><div><p className="eyebrow">{t.trend}</p><h3>{labels.trendTitle}</h3></div><span>{detail.checkins.length} {labels.checkins}</span></div><Trend rows={detail.checkins}/></section>
            <section className="clinical-section"><div className="section-heading"><div><p className="eyebrow">{t.timeline}</p><h3>{labels.trailTitle}</h3></div><span>{detail.checkins.length} {labels.checkins}</span></div><div className="daily-record-grid">{detail.checkins.map((row, index) => <article id={'day-' + row.day} className={'daily-record ' + (detail.assessments[index]?.tier || 'L0')} key={row.day}><header><div><strong>D{row.day}</strong><span>{clinical.latest}</span></div><Tier value={detail.assessments[index]?.tier || 'L0'} compact/></header><div className="record-values"><div><span>{t.weight}</span><b>{row.weight_kg} kg</b></div><div><span>SBP</span><b>{row.sbp}</b></div><div><span>HR</span><b>{row.hr}</b></div><div><span>{t.oxygen}</span><b className={row.spo2 < 90 ? 'critical' : ''}>{row.spo2}%</b></div><div><span>{clinical.edema}</span><b>{row.edema_grade}</b></div></div></article>)}</div></section>            <section className="clinical-section evidence-section"><div className="section-heading"><div><p className="eyebrow">{t.evidence}</p><h3>{labels.rationaleTitle}</h3></div><span>{evidence.length} {labels.flagged}</span></div><div className="evidence-list">{evidence.length ? evidence.map(assessment => <article className={'evidence-item ' + assessment.tier} key={assessment.day}><div><span>Day {assessment.day}</span><Tier value={assessment.tier}/></div><div><strong>{assessment.summary}</strong><ul>{assessment.fired_rules.map(rule => <li key={rule.rule_id}><code>{rule.rule_id}</code>{rule.label}</li>)}</ul></div></article>) : <p className="no-evidence">{t.noEvidence}</p>}</div></section>
          </> : <div className="empty-state"><span className="empty-mark"><Icon name="workspace"/></span><h2>{t.workspace}</h2><p>{t.empty}</p></div>}</section>
          <aside className="decision-panel">{detail ? <><div className="panel-heading"><p className="eyebrow">{t.agent}</p><h2>{latest?.tier || 'L0'}</h2><p>{latest?.summary}</p></div><form onSubmit={submit}><label>{t.tier}<select value={tier} onChange={event => setTier(event.target.value)}>{['L0', 'L1', 'L2', 'L3'].map(value => <option key={value}>{value}</option>)}</select></label><label>{t.assessment}<select value={agreement} onChange={event => setAgreement(event.target.value)}><option value="agree">{t.agree}</option><option value="modify">{t.modify}</option><option value="disagree">{t.disagree}</option></select></label><label>{t.note}<textarea value={note} required={agreement !== 'agree'} onChange={event => setNote(event.target.value)} placeholder={agreement === 'agree' ? 'Optional' : 'Required'}/></label><button className="primary-button" disabled={saving}>{saving ? t.saving : t.save}</button>{message && <p className="save-message">{message}</p>}</form><section className="llm-review"><div className="llm-review-heading"><div><Icon name="shield"/><div><span>{t.llm}</span><b>{llmStatus?.enabled ? t.llmEnabled : t.llmDisabled}</b></div></div></div>{llmStatus?.enabled ? <button type="button" className="llm-button" disabled={llmLoading} onClick={requestLLMReview}>{llmLoading ? t.llmRunning : t.llmRun}</button> : <p className="llm-disabled-copy">{llmStatus?.message}</p>}{llmError && <p className="llm-error" role="alert">{llmError}</p>}{llmAssessment && <div className="llm-output"><div><span>{t.risk}</span><Tier value={llmAssessment.risk_tier}/></div><p>{llmAssessment.rationale}</p><div className="ai-assessment-meta"><span>{lab.confidence}: <b>{llmAssessment.confidence}</b></span><span>{lab.alignment}: <b>{llmAssessment.rules_alignment === 'rules_align' ? lab.rulesAlign : llmAssessment.rules_alignment === 'rules_differ' ? lab.rulesDiffer : lab.insufficient}</b></span></div>{llmAssessment.evidence?.length > 0 && <><strong>{lab.evidence}</strong><div className="ai-evidence">{llmAssessment.evidence.map(citation => <button type="button" key={citation.day + citation.field} onClick={() => jumpToDay(citation.day)} title={citation.note}><b>D{citation.day}</b><span>{citation.field}</span><em>{citation.value}</em></button>)}</div></>}<strong>{t.llmSignals}</strong><ul>{llmAssessment.key_signals.map(signal => <li key={signal}>{signal}</li>)}</ul><strong>{t.llmQuestions}</strong><ul>{llmAssessment.questions.map(question => <li key={question}>{question}</li>)}</ul><small>{llmAssessment.safety_note || t.llmResearchOnly}</small></div>}</section><p className="panel-footer"><Icon name="shield"/>{t.synthetic}</p></> : <div className="decision-empty"><Icon name="workspace"/><p>{t.decision}</p></div>}</aside>
        </div>
      </section>}
    </section>
    {locked && <div className="lock-screen"><div><span className="lock-icon"><Icon name="lock"/></span><h2>{t.locked}</h2><p>{t.lockText}</p><button className="primary-button" onClick={() => { setLocked(false); setActivityAt(Date.now()) }}>{t.unlock}</button></div></div>}
  </main>
}
createRoot(document.getElementById('root')!).render(<App/>)
