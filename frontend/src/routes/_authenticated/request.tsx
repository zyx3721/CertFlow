import { createFileRoute } from '@tanstack/react-router';
import { RequestPage } from '@/features/pki/RequestPage';

export const Route = createFileRoute('/_authenticated/request')({ component: RequestPage });
