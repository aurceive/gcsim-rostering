import { Badge } from "@gcsim/primitives";
import { memo } from "react";
import { useTranslation } from "react-i18next";
import { cn } from "../lib/utils";

type Props = {
	commit?: string;
	className?: string;
};

export const Commit = memo(({ commit, className }: Props) => {
	const { t } = useTranslation();
	if (!commit) return null;
	return (
		<Badge className={cn("text-g-sm font-g-mono", className)}>
			<span className="flex flex-row items-center gap-2">
				<span className="text-g-ink-mute">{t("result.metadata_commit")}</span>
				<span className="font-bold" title={commit}>
					{commit.substring(0, 7)}
				</span>
			</span>
		</Badge>
	);
});
