package service

import (
	"context"
	"crypto/md5"
	"fmt"
	"strconv"
	"strings"

	"github.com/kolide/fleet/server/contexts/viewer"
	"github.com/kolide/fleet/server/kolide"
	"github.com/pkg/errors"
)

func (svc service) ImportConfig(ctx context.Context, cfg *kolide.ImportConfig) (*kolide.ImportConfigResponse, error) {
	resp := &kolide.ImportConfigResponse{
		ImportStatusBySection: make(map[kolide.ImportSection]*kolide.ImportStatus),
	}
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, errors.New("internal error, unable to fetch user")
	}
	tx, err := svc.ds.Begin()
	if err != nil {
		return nil, err
	}

	if err := svc.importOptions(cfg.Options, resp, tx); err != nil {
		svc.rollbackImportConfig(tx, "importOptions")
		return nil, errors.Wrap(err, "importOptions failed")
	}
	if err := svc.importPacks(vc.UserID(), cfg, resp, tx); err != nil {
		svc.rollbackImportConfig(tx, "importPacks")
		return nil, errors.Wrap(err, "importPacks failed")
	}
	if err := svc.importScheduledQueries(vc.UserID(), cfg, resp, tx); err != nil {
		svc.rollbackImportConfig(tx, "importScheduledQueries")
		return nil, errors.Wrap(err, "importScheduledQueries failed")
	}
	if err := svc.importDecorators(cfg, resp, tx); err != nil {
		svc.rollbackImportConfig(tx, "importDecorators")
		return nil, errors.Wrap(err, "importDecorators")
	}
	if err := svc.importFIMSections(cfg, resp, tx); err != nil {
		svc.rollbackImportConfig(tx, "importFIMSections")
		return nil, errors.Wrap(err, "importFIMSections")
	}
	if cfg.DryRun {
		if err := tx.Rollback(); err != nil {
			return nil, errors.Wrap(err, "dry run rollback failed")
		}
		return resp, nil
	}
	if err := tx.Commit(); err != nil {
		return nil, errors.Wrap(err, "commit failed")
	}
	return resp, nil
}

func (svc service) rollbackImportConfig(tx kolide.Transaction, method string) {
	if err := tx.Rollback(); err != nil {
		svc.logger.Log(
			"method", method,
			"err", errors.Wrap(err, fmt.Sprintf("db rollback failed in %s", method)),
		)
	}
}

func (svc service) importYARA(cfg *kolide.ImportConfig, resp *kolide.ImportConfigResponse, tx kolide.Transaction) error {
	if cfg.YARA != nil {
		for sig, paths := range cfg.YARA.Signatures {
			ysg := &kolide.YARASignatureGroup{
				SignatureName: sig,
				Paths:         paths,
			}
			_, err := svc.ds.NewYARASignatureGroup(ysg, kolide.HasTransaction(tx))
			if _, ok := err.(dbDuplicateError); ok {
				resp.Status(kolide.YARAFileSection).SkipCount++
				resp.Status(kolide.YARAFileSection).Warning(kolide.YARADuplicate, "skipped '%s', already exists", sig)
				continue
			}
			if err != nil {
				return err
			}
			resp.Status(kolide.YARASigSection).ImportCount++
			resp.Status(kolide.YARASigSection).Message("imported '%s'", sig)
		}
		for section, sigs := range cfg.YARA.FilePaths {
			for _, sig := range sigs {
				err := svc.ds.NewYARAFilePath(section, sig, kolide.HasTransaction(tx))
				if _, ok := err.(dbDuplicateError); ok {
					resp.Status(kolide.YARAFileSection).SkipCount++
					resp.Status(kolide.YARAFileSection).Warning(kolide.YARADuplicate, "skipped '%s', already exists", section)
					continue
				}
				if err != nil {
					return err
				}
				resp.Status(kolide.YARAFileSection).ImportCount++
				resp.Status(kolide.YARAFileSection).Message("imported '%s'", section)
			}
		}
	}
	return nil
}

type dbDuplicateError interface {
	IsExists() bool
}

func (svc service) importFIMSections(cfg *kolide.ImportConfig, resp *kolide.ImportConfigResponse, tx kolide.Transaction) error {
	if cfg.FileIntegrityMonitoring != nil {
		for sectionName, paths := range cfg.FileIntegrityMonitoring {
			fp := &kolide.FIMSection{
				SectionName: sectionName,
				Description: "imported",
				Paths:       paths,
			}
			_, err := svc.ds.NewFIMSection(fp, kolide.HasTransaction(tx))
			if _, ok := err.(dbDuplicateError); ok {
				resp.Status(kolide.FilePathsSection).SkipCount++
				resp.Status(kolide.FilePathsSection).Warning(kolide.FIMDuplicate, "skipped '%s', already exists", sectionName)
				continue
			}
			if err != nil {
				return err
			}
			resp.Status(kolide.FilePathsSection).ImportCount++
			resp.Status(kolide.FilePathsSection).Message("imported '%s'", sectionName)
		}
	}
	// this has to happen AFTER fim section, because it requires file paths
	return svc.importYARA(cfg, resp, tx)
}

func (svc service) getExistingDecoratorQueries(tx kolide.Transaction) (map[string]int, error) {
	decs, err := svc.ds.ListDecorators(kolide.HasTransaction(tx))
	if err != nil {
		return nil, err
	}
	queryHashes := map[string]int{}
	for _, dec := range decs {
		hash := fmt.Sprintf("%x", md5.Sum([]byte(dec.Query)))
		queryHashes[hash] = 0
	}
	return queryHashes, nil
}

func decoratorExists(query string, queryHashes map[string]int) bool {
	hash := fmt.Sprintf("%x", md5.Sum([]byte(query)))
	_, exists := queryHashes[hash]
	return exists
}

func (svc service) importDecorators(cfg *kolide.ImportConfig, resp *kolide.ImportConfigResponse, tx kolide.Transaction) error {
	if cfg.Decorators != nil {
		queryHashes, err := svc.getExistingDecoratorQueries(tx)
		if err != nil {
			return errors.Wrap(err, "getting existing queries")
		}

		for _, query := range cfg.Decorators.Load {
			if decoratorExists(query, queryHashes) {
				resp.Status(kolide.DecoratorsSection).SkipCount++
				resp.Status(kolide.DecoratorsSection).Warning(kolide.QueryDuplicate, "skipped load '%s'", query)
				continue
			}
			decName, err := uniqueImportName()
			if err != nil {
				return err
			}
			decorator := &kolide.Decorator{
				Name:  decName,
				Query: query,
				Type:  kolide.DecoratorLoad,
			}
			_, err = svc.ds.NewDecorator(decorator, kolide.HasTransaction(tx))
			if err != nil {
				return err
			}
			resp.Status(kolide.DecoratorsSection).ImportCount++
			resp.Status(kolide.DecoratorsSection).Warning("imported load '%s'", query)
		}
		for _, query := range cfg.Decorators.Always {
			if decoratorExists(query, queryHashes) {
				resp.Status(kolide.DecoratorsSection).SkipCount++
				resp.Status(kolide.DecoratorsSection).Warning(kolide.QueryDuplicate, "skipped always '%s'", query)
				continue
			}
			decName, err := uniqueImportName()
			if err != nil {
				return err
			}
			decorator := &kolide.Decorator{
				Name:  decName,
				Query: query,
				Type:  kolide.DecoratorAlways,
			}
			_, err = svc.ds.NewDecorator(decorator, kolide.HasTransaction(tx))
			if err != nil {
				return err
			}
			resp.Status(kolide.DecoratorsSection).ImportCount++
			resp.Status(kolide.DecoratorsSection).Message("imported always '%s'", query)
		}

		for key, queries := range cfg.Decorators.Interval {
			for _, query := range queries {
				if decoratorExists(query, queryHashes) {
					resp.Status(kolide.DecoratorsSection).SkipCount++
					resp.Status(kolide.DecoratorsSection).Warning(kolide.QueryDuplicate, "skipped interval '%s'", query)
					continue
				}
				interval, err := strconv.ParseInt(key, 10, 32)
				if err != nil {
					return err
				}
				decName, err := uniqueImportName()
				if err != nil {
					return err
				}
				decorator := &kolide.Decorator{
					Name:     decName,
					Query:    query,
					Type:     kolide.DecoratorInterval,
					Interval: uint(interval),
				}
				_, err = svc.ds.NewDecorator(decorator, kolide.HasTransaction(tx))
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

func (svc service) importScheduledQueries(uid uint, cfg *kolide.ImportConfig, resp *kolide.ImportConfigResponse, tx kolide.Transaction) error {
	_, ok, err := svc.ds.PackByName(kolide.ImportPackName, kolide.HasTransaction(tx))
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
	pack, err = svc.ds.NewPack(pack, kolide.HasTransaction(tx))
	if err != nil {
		return err
	}
	resp.Status(kolide.PacksSection).ImportCount++
	resp.Status(kolide.PacksSection).Message("created import pack")

	for queryName, queryDetails := range cfg.Schedule {
		var query *kolide.Query
		query, ok, err = svc.ds.QueryByName(queryName, kolide.HasTransaction(tx))
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
			query, err = svc.ds.NewQuery(query, kolide.HasTransaction(tx))
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
			Interval: uint(queryDetails.Interval),
			Snapshot: queryDetails.Snapshot,
			Removed:  queryDetails.Removed,
			Platform: queryDetails.Platform,
			Version:  queryDetails.Version,
			Shard:    configInt2Ptr(queryDetails.Shard),
		}
		_, err = svc.ds.NewScheduledQuery(sq, kolide.HasTransaction(tx))
		if err != nil {
			return nil
		}
		resp.Status(kolide.PacksSection).Message(
			"added query '%s' to '%s'", query.Name, pack.Name,
		)
	}
	return nil
}

func (svc service) importPacks(uid uint, cfg *kolide.ImportConfig, resp *kolide.ImportConfigResponse, tx kolide.Transaction) error {
	labelCache := map[string]*kolide.Label{}
	packs, err := cfg.CollectPacks()
	if err != nil {
		return err
	}
	for packName, packDetails := range packs {
		_, ok, err := svc.ds.PackByName(packName, kolide.HasTransaction(tx))
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
		pack, err = svc.ds.NewPack(pack, kolide.HasTransaction(tx))
		if err != nil {
			return err
		}
		err = svc.createLabelsForPack(pack, &packDetails, labelCache, resp, tx)
		if err != nil {
			return err
		}
		err = svc.createQueriesForPack(uid, pack, &packDetails, resp, tx)
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
	random, err := kolide.RandomText(6)
	if err != nil {
		return "", err
	}
	return "import_" + random, nil
}

func (svc service) createQueriesForPack(uid uint, pack *kolide.Pack, details *kolide.PackDetails,
	resp *kolide.ImportConfigResponse, tx kolide.Transaction) error {
	for queryName, queryDetails := range details.Queries {
		query, ok, err := svc.ds.QueryByName(queryName, kolide.HasTransaction(tx))
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
			query, err = svc.ds.NewQuery(query, kolide.HasTransaction(tx))
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
			Interval: uint(queryDetails.Interval),
			Platform: queryDetails.Platform,
			Snapshot: queryDetails.Snapshot,
			Removed:  queryDetails.Removed,
			Version:  queryDetails.Version,
			Shard:    configInt2Ptr(queryDetails.Shard),
		}
		_, err = svc.ds.NewScheduledQuery(scheduledQuery, kolide.HasTransaction(tx))
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
	cache map[string]*kolide.Label, resp *kolide.ImportConfigResponse, tx kolide.Transaction) error {
	for _, query := range details.Discovery {
		hash := hashQuery(details.Platform, query)
		label, ok := cache[hash]
		// add existing label to pack
		if ok {
			err := svc.ds.AddLabelToPack(label.ID, pack.ID, kolide.HasTransaction(tx))
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
		label, err = svc.ds.NewLabel(label, kolide.HasTransaction(tx))
		if err != nil {
			return err
		}
		// hang on to label so we can reuse it for other packs if needed
		cache[hash] = label
		err = svc.ds.AddLabelToPack(label.ID, pack.ID, kolide.HasTransaction(tx))
		if err != nil {
			return err
		}
		resp.Status(kolide.PacksSection).Message(
			"added label '%s' to '%s'", label.Name, pack.Name,
		)
	}
	return nil
}

func (svc service) importOptions(opts kolide.OptionNameToValueMap, resp *kolide.ImportConfigResponse, tx kolide.Transaction) error {
	var updateOptions []kolide.Option
	for optName, optValue := range opts {
		opt, err := svc.ds.OptionByName(optName, kolide.HasTransaction(tx))
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
		if err := svc.ds.SaveOptions(updateOptions, kolide.HasTransaction(tx)); err != nil {
			return err
		}
	}
	return nil
}

func configInt2Ptr(ci *kolide.OsQueryConfigInt) *uint {
	if ci == nil {
		return nil
	}
	ui := uint(*ci)
	return &ui
}
