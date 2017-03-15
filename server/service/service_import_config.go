package service

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/kolide/kolide/server/contexts/viewer"
	"github.com/kolide/kolide/server/kolide"
)

func (svc service) ImportConfig(ctx context.Context, cfg *kolide.ImportConfig) (*kolide.ImportConfigResponse, error) {
	resp := &kolide.ImportConfigResponse{
		ImportStatusBySection: make(map[kolide.ImportSection]*kolide.ImportStatus),
	}
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, errors.New("internal error, unable to fetch user")
	}
	if err := svc.importOptions(cfg.Options, resp); err != nil {
		return nil, err
	}
	if err := svc.importPacks(vc.UserID(), cfg, resp); err != nil {
		return nil, err
	}
	if err := svc.importScheduledQueries(vc.UserID(), cfg, resp); err != nil {
		return nil, err
	}
	if err := svc.importDecorators(cfg, resp); err != nil {
		return nil, err
	}
	if err := svc.importFIMSections(cfg, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (svc service) importYARA(cfg *kolide.ImportConfig, resp *kolide.ImportConfigResponse) error {
	if cfg.YARA != nil {
		for sig, paths := range cfg.YARA.Signatures {
			ysg := &kolide.YARASignatureGroup{
				SignatureName: sig,
				Paths:         paths,
			}
			_, err := svc.ds.NewYARASignatureGroup(ysg)
			if err != nil {
				return err
			}
			resp.Status(kolide.YARASigSection).ImportCount++
			resp.Status(kolide.YARASigSection).Message("imported '%s'", sig)
		}
		for section, sigs := range cfg.YARA.FilePaths {
			for _, sig := range sigs {
				err := svc.ds.NewYARAFilePath(section, sig)
				if err != nil {
					return err
				}
			}
			resp.Status(kolide.YARAFileSection).ImportCount++
			resp.Status(kolide.YARAFileSection).Message("imported '%s'", section)
		}
	}
	return nil
}

func (svc service) importFIMSections(cfg *kolide.ImportConfig, resp *kolide.ImportConfigResponse) error {
	if cfg.FileIntegrityMonitoring != nil {
		for sectionName, paths := range cfg.FileIntegrityMonitoring {
			fp := &kolide.FIMSection{
				SectionName: sectionName,
				Description: "imported",
				Paths:       paths,
			}
			_, err := svc.ds.NewFIMSection(fp)
			if err != nil {
				return err
			}
			resp.Status(kolide.FilePathsSection).ImportCount++
			resp.Status(kolide.FilePathsSection).Message("imported '%s'", sectionName)
		}
	}
	// this has to happen AFTER fim section, because it requires file paths
	return svc.importYARA(cfg, resp)
}

func (svc service) importDecorators(cfg *kolide.ImportConfig, resp *kolide.ImportConfigResponse) error {
	if cfg.Decorators != nil {
		for _, query := range cfg.Decorators.Load {
			decorator := &kolide.Decorator{
				Query: query,
				Type:  kolide.DecoratorLoad,
			}
			_, err := svc.ds.NewDecorator(decorator)
			if err != nil {
				return err
			}
			resp.Status(kolide.DecoratorsSection).ImportCount++
			resp.Status(kolide.DecoratorsSection).Message("imported load '%s'", query)
		}
		for _, query := range cfg.Decorators.Always {
			decorator := &kolide.Decorator{
				Query: query,
				Type:  kolide.DecoratorAlways,
			}
			_, err := svc.ds.NewDecorator(decorator)
			if err != nil {
				return err
			}
			resp.Status(kolide.DecoratorsSection).ImportCount++
			resp.Status(kolide.DecoratorsSection).Message("imported always '%s'", query)
		}
		for key, queries := range cfg.Decorators.Interval {
			for _, query := range queries {
				interval, err := strconv.ParseInt(key, 10, 32)
				if err != nil {
					return err
				}
				decorator := &kolide.Decorator{
					Query:    query,
					Type:     kolide.DecoratorInterval,
					Interval: uint(interval),
				}
				_, err = svc.ds.NewDecorator(decorator)
				if err != nil {
					return err
				}
				resp.Status(kolide.DecoratorsSection).ImportCount++
				resp.Status(kolide.DecoratorsSection).Message("imported interval %d '%s'", interval, query)
			}
		}

	}
	return nil
}

func (svc service) importScheduledQueries(uid uint, cfg *kolide.ImportConfig, resp *kolide.ImportConfigResponse) error {
	_, ok, err := svc.ds.PackByName(kolide.ImportPackName)
	if ok {
		resp.Status(kolide.PacksSection).Warning(
			kolide.PackDuplicate, "skipped '%s' already exists", kolide.ImportPackName,
		)
		resp.Status(kolide.PacksSection).SkipCount++
		return nil
	}
	// create import pack to hold imported scheduled queries
	pack := &kolide.Pack{
		Name:        kolide.ImportPackName,
		Description: "holds imported scheduled queries",
		CreatedBy:   uid,
		Disabled:    false,
	}
	pack, err = svc.ds.NewPack(pack)
	if err != nil {
		return err
	}
	resp.Status(kolide.PacksSection).ImportCount++
	resp.Status(kolide.PacksSection).Message("created import pack")

	for queryName, queryDetails := range cfg.Schedule {
		var query *kolide.Query
		query, ok, err = svc.ds.QueryByName(queryName)
		// if we find the query check to see if the import query matches the
		// query we have, if it doesn't skip it
		if ok {
			if hashQuery("", query.Query) != hashQuery("", queryDetails.Query) {
				resp.Status(kolide.PacksSection).Warning(
					kolide.DifferentQuerySameName,
					"queries named '%s' have different statements and won't be added to '%s'",
					queryName,
					pack.Name,
				)
				continue
			}
			resp.Status(kolide.QueriesSection).Warning(
				kolide.QueryDuplicate, "skipped '%s' different query of same name already exists", queryName,
			)
			resp.Status(kolide.QueriesSection).SkipCount++
		} else {
			// if query doesn't exist, create it
			query = &kolide.Query{
				Name:        queryName,
				Description: "imported",
				Query:       queryDetails.Query,
				Saved:       true,
				AuthorID:    uid,
			}
			query, err = svc.ds.NewQuery(query)
			if err != nil {
				return err
			}
			resp.Status(kolide.QueriesSection).ImportCount++
			resp.Status(kolide.QueriesSection).Message(
				"imported scheduled query '%s'", query.Name,
			)
		}
		sq := &kolide.ScheduledQuery{
			PackID:   pack.ID,
			QueryID:  query.ID,
			Interval: queryDetails.Interval,
			Snapshot: queryDetails.Snapshot,
			Removed:  queryDetails.Removed,
			Platform: queryDetails.Platform,
			Version:  queryDetails.Version,
			Shard:    queryDetails.Shard,
		}
		_, err = svc.ds.NewScheduledQuery(sq)
		if err != nil {
			return nil
		}
		resp.Status(kolide.PacksSection).Message(
			"added query '%s' to '%s'", query.Name, pack.Name,
		)
	}
	return nil
}

func (svc service) importPacks(uid uint, cfg *kolide.ImportConfig, resp *kolide.ImportConfigResponse) error {
	labelCache := map[string]*kolide.Label{}
	packs, err := cfg.CollectPacks()
	if err != nil {
		return err
	}
	for packName, packDetails := range packs {
		_, ok, err := svc.ds.PackByName(packName)
		if err != nil {
			return err
		}
		if ok {
			resp.Status(kolide.PacksSection).Warning(
				kolide.PackDuplicate, "skipped '%s' already exists", packName,
			)
			resp.Status(kolide.PacksSection).SkipCount++
			continue
		}
		// import new pack
		if packDetails.Shard != nil {
			resp.Status(kolide.PacksSection).Warning(
				kolide.Unsupported,
				"shard for pack '%s'",
				packName,
			)
		}
		if packDetails.Version != nil {
			resp.Status(kolide.PacksSection).Warning(
				kolide.Unsupported,
				"version for pack '%s'",
				packName,
			)
		}
		pack := &kolide.Pack{
			Name:        packName,
			Description: "Imported pack",
			Platform:    packDetails.Platform,
		}
		pack, err = svc.ds.NewPack(pack)
		if err != nil {
			return err
		}
		err = svc.createLabelsForPack(pack, &packDetails, labelCache, resp)
		if err != nil {
			return err
		}
		err = svc.createQueriesForPack(uid, pack, &packDetails, resp)
		if err != nil {
			return err
		}
		resp.Status(kolide.PacksSection).ImportCount++
		resp.Status(kolide.PacksSection).Message("imported '%s'", packName)
	}
	return nil
}

func hashQuery(platform, query string) string {
	s := strings.Replace(query, " ", "", -1)
	s = strings.Replace(s, "\t", "", -1)
	s = strings.Replace(s, "\n", "", -1)
	s = strings.Trim(s, ";")
	s = platform + s
	return strings.ToLower(s)
}

func uniqueImportName() (string, error) {
	random, err := kolide.RandomText(12)
	if err != nil {
		return "", err
	}
	return "import_" + random, nil
}

func (svc service) createQueriesForPack(uid uint, pack *kolide.Pack, details *kolide.PackDetails,
	resp *kolide.ImportConfigResponse) error {
	for queryName, queryDetails := range details.Queries {
		query, ok, err := svc.ds.QueryByName(queryName)
		if err != nil {
			return err
		}
		// if the query isn't already in the database, create it
		if !ok {
			query = &kolide.Query{
				Name:        queryName,
				Description: "imported",
				Query:       queryDetails.Query,
				Saved:       true,
				AuthorID:    uid,
			}
			query, err = svc.ds.NewQuery(query)
			if err != nil {
				return err
			}
			resp.Status(kolide.QueriesSection).Message(
				"created '%s' as part of pack '%s'", queryName, pack.Name,
			)
			resp.Status(kolide.QueriesSection).ImportCount++
		}
		// associate query with pack
		scheduledQuery := &kolide.ScheduledQuery{
			PackID:   pack.ID,
			QueryID:  query.ID,
			Interval: queryDetails.Interval,
			Platform: queryDetails.Platform,
			Snapshot: queryDetails.Snapshot,
			Removed:  queryDetails.Removed,
			Version:  queryDetails.Version,
			Shard:    queryDetails.Shard,
		}
		_, err = svc.ds.NewScheduledQuery(scheduledQuery)
		if err != nil {
			return nil
		}
		resp.Status(kolide.PacksSection).Message("added query '%s'", query.Name)

	}
	return nil
}

// createLabelsForPack Iterates through discover queries, creates a label for
// each query and assigns it to the pack passed as an argument.  Once a Label is created we cache
// it for reuse.
func (svc service) createLabelsForPack(pack *kolide.Pack, details *kolide.PackDetails,
	cache map[string]*kolide.Label, resp *kolide.ImportConfigResponse) error {
	for _, query := range details.Discovery {
		hash := hashQuery(details.Platform, query)
		label, ok := cache[hash]
		// add existing label to pack
		if ok {
			err := svc.ds.AddLabelToPack(label.ID, pack.ID)
			if err != nil {
				return err
			}
			resp.Status(kolide.PacksSection).Message(
				"added label '%s' to pack '%s'", label.Name, pack.Name,
			)
			continue
		}
		// create new label and add it to pack
		labelName, err := uniqueImportName()
		if err != nil {
			return err
		}
		label = &kolide.Label{
			Name:        labelName,
			Query:       query,
			Description: "imported",
			LabelType:   kolide.LabelTypeRegular,
			Platform:    details.Platform,
		}
		label, err = svc.ds.NewLabel(label)
		if err != nil {
			return err
		}
		// hang on to label so we can reuse it for other packs if needed
		cache[hash] = label
		err = svc.ds.AddLabelToPack(label.ID, pack.ID)
		if err != nil {
			return err
		}
		resp.Status(kolide.PacksSection).Message(
			"added label '%s' to '%s'", label.Name, pack.Name,
		)
	}
	return nil
}

func (svc service) importOptions(opts kolide.OptionNameToValueMap, resp *kolide.ImportConfigResponse) error {
	var updateOptions []kolide.Option
	for optName, optValue := range opts {
		opt, err := svc.ds.OptionByName(optName)
		if err != nil {
			resp.Status(kolide.OptionsSection).Warning(
				kolide.OptionUnknown, "skipped '%s' can't find option", optName,
			)
			resp.Status(kolide.OptionsSection).SkipCount++
			continue
		}
		if opt.ReadOnly {
			resp.Status(kolide.OptionsSection).Warning(
				kolide.OptionReadonly, "skipped '%s' can't change read only option", optName,
			)
			resp.Status(kolide.OptionsSection).SkipCount++
			continue
		}
		if opt.OptionSet() {
			resp.Status(kolide.OptionsSection).Warning(
				kolide.OptionAlreadySet, "skipped '%s' can't change option that is already set", optName,
			)
			resp.Status(kolide.OptionsSection).SkipCount++
			continue
		}
		opt.SetValue(optValue)
		resp.Status(kolide.OptionsSection).Message("set %s value to %v", optName, optValue)
		resp.Status(kolide.OptionsSection).ImportCount++
		updateOptions = append(updateOptions, *opt)
	}
	if len(updateOptions) > 0 {
		if err := svc.ds.SaveOptions(updateOptions); err != nil {
			return err
		}
	}
	return nil
}
