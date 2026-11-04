import { Input, SegmentedControl } from "@mantine/core";

/**
 * 少ない選択肢から 1 つ選ぶ。横一列の切り替えにして、狭い列にも 1 行で収める。
 * pointer で選んだときは focus を外し、矢印 key を page の操作に返す（Tab で入ったときは矢印で選べるまま）。
 * 矢印 key で radio を選んだときの click は detail が 0 なので、それで pointer と見分ける
 */
export const Choice = ({
  label,
  data,
  value,
  onChange,
}: {
  label: string;
  data: { value: string; label: string }[];
  value: string;
  onChange: (v: string) => void;
}) => (
  <Input.Wrapper
    label={label}
    size="xs"
    onClick={(e) => {
      // label の click は radio へ渡り、そこで focus が移る。それが済んでから外す
      if (e.detail > 0)
        setTimeout(() => {
          if (document.activeElement instanceof HTMLElement) document.activeElement.blur();
        });
    }}
  >
    <SegmentedControl fullWidth size="xs" data={data} value={value} onChange={onChange} />
  </Input.Wrapper>
);
