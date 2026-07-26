import { createFileRoute } from '@tanstack/react-router';
import { WorkflowsPage } from '@/features/pki/WorkflowsPage';

export const Route = createFileRoute('/_authenticated/workflows')({ component: WorkflowsPage });
