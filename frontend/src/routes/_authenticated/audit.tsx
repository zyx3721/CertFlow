import { createFileRoute } from '@tanstack/react-router';
import { AuditPage } from '@/features/pki/AuditPage';

export const Route = createFileRoute('/_authenticated/audit')({ component: AuditPage });
