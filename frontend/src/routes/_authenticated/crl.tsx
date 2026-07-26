import { createFileRoute } from '@tanstack/react-router';
import { CRLPage } from '@/features/pki/CRLPage';

export const Route = createFileRoute('/_authenticated/crl')({ component: CRLPage });
