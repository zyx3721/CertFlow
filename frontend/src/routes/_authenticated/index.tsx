import { createFileRoute } from '@tanstack/react-router';
import { DashboardPage } from '@/features/pki/DashboardPage';

export const Route = createFileRoute('/_authenticated/')({ component: DashboardPage });
