import { createFileRoute } from '@tanstack/react-router';
import { CAPage } from '@/features/pki/CAPage';

export const Route = createFileRoute('/_authenticated/cas')({ component: CAPage });
