import { Input } from '@/components/ui/input';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';

const validityOptions = [
  ['365', '1 年'],
  ['730', '2 年'],
  ['1095', '3 年'],
  ['1825', '5 年'],
  ['3650', '10 年'],
  ['7300', '20 年'],
] as const;

export const customValidityOption = 'custom';
const customValidityUnits = [
  ['day', '天', 1],
  ['month', '月', 30],
  ['year', '年', 365],
] as const;

export type CustomValidityUnit = (typeof customValidityUnits)[number][0];

export function requestedValidityDays(
  option: string,
  customValue: string,
  customUnit: CustomValidityUnit
) {
  if (option !== customValidityOption) return Number(option);
  const unit = customValidityUnits.find(item => item[0] === customUnit);
  return Number(customValue) * (unit?.[2] ?? 1);
}

export function validityLabel(
  option: string,
  customValue: string,
  customUnit: CustomValidityUnit
) {
  if (option !== customValidityOption) {
    return validityOptions.find(item => item[0] === option)?.[1] ?? `${option} 天`;
  }
  const unit = customValidityUnits.find(item => item[0] === customUnit);
  return `${customValue || '-'} ${unit?.[1] ?? '天'}`;
}

export function RequestValidityField({
  option,
  customValue,
  customUnit,
  onOptionChange,
  onCustomValueChange,
  onCustomUnitChange,
}: {
  option: string;
  customValue: string;
  customUnit: CustomValidityUnit;
  onOptionChange: (value: string) => void;
  onCustomValueChange: (value: string) => void;
  onCustomUnitChange: (value: CustomValidityUnit) => void;
}) {
  const custom = option === customValidityOption;
  return (
    <div
      className={
        custom
          ? 'grid grid-cols-[minmax(0,0.8fr)_minmax(92px,0.55fr)_minmax(88px,0.45fr)] gap-2'
          : ''
      }
    >
      <Select value={option} onValueChange={onOptionChange}>
        <SelectTrigger className="font-normal">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {validityOptions.map(([value, label]) => (
            <SelectItem key={value} value={value} className="font-normal">
              {label}
            </SelectItem>
          ))}
          <SelectItem value={customValidityOption} className="font-normal">
            自定义
          </SelectItem>
        </SelectContent>
      </Select>
      {custom ? (
        <>
          <Input
            value={customValue}
            onChange={event => onCustomValueChange(event.target.value)}
            inputMode="numeric"
            placeholder="填写数字"
            aria-label="自定义有效期数值"
          />
          <Select value={customUnit} onValueChange={onCustomUnitChange}>
            <SelectTrigger className="font-normal">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {customValidityUnits.map(([value, label]) => (
                <SelectItem key={value} value={value} className="font-normal">
                  {label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </>
      ) : null}
    </div>
  );
}
