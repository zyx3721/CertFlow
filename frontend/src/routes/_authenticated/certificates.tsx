import { createFileRoute } from '@tanstack/react-router';
import { CertificatesPage } from '@/features/pki/CertificatesPage';

export const Route = createFileRoute('/_authenticated/certificates')({
  component: CertificatesPage,
});
